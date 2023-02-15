/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package createworker

import (
	_ "embed"
	b64 "encoding/base64"
	"encoding/json"
	"net/url"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

//go:embed files/gcp-compute-persistent-disk-csi-driver.yaml
var csiManifest string

type GCPBuilder struct {
	capxProvider string
	capxName     string
	capxTemplate string
	capxEnvVars  []string
	storageClass string
	csiNamespace string
}

func newGCPBuilder() *GCPBuilder {
	return &GCPBuilder{}
}

func (b *GCPBuilder) setCapx(managed bool) {
	b.capxProvider = "gcp:v1.2.1"
	b.capxName = "capg"
	b.storageClass = "csi-gcp-pd"
	if managed {
		b.capxTemplate = "gcp.gke.tmpl"
		b.csiNamespace = ""
	} else {
		b.capxTemplate = "gcp.tmpl"
		b.csiNamespace = "gce-pd-csi-driver"
	}
}

func (b *GCPBuilder) setCapxEnvVars(p ProviderParams) {
	data := map[string]interface{}{
		"type":                        "service_account",
		"project_id":                  p.credentials["ProjectID"],
		"private_key_id":              p.credentials["PrivateKeyID"],
		"private_key":                 p.credentials["PrivateKey"],
		"client_email":                p.credentials["ClientEmail"],
		"client_id":                   p.credentials["ClientID"],
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://accounts.google.com/o/oauth2/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/" + url.QueryEscape(p.credentials["ClientEmail"]),
	}

	jsonData, _ := json.Marshal(data)
	b.capxEnvVars = []string{
		"GCP_B64ENCODED_CREDENTIALS=" + b64.StdEncoding.EncodeToString([]byte(jsonData)),
		"GITHUB_TOKEN=" + p.githubToken,
	}
}

func (b *GCPBuilder) getProvider() Provider {
	return Provider{
		capxProvider: b.capxProvider,
		capxName:     b.capxName,
		capxTemplate: b.capxTemplate,
		capxEnvVars:  b.capxEnvVars,
		storageClass: b.storageClass,
		csiNamespace: b.csiNamespace,
	}
}

func (b *GCPBuilder) installCSI(n nodes.Node, k string) error {
	var c string
	var err error

	// Create CSI namespace
	c = "kubectl --kubeconfig " + k + " create namespace " + b.csiNamespace
	err = executeCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create CSI namespace")
	}

	// Create CSI secret in CSI namespace
	secret, _ := b64.StdEncoding.DecodeString(strings.Split(b.capxEnvVars[0], "GCP_B64ENCODED_CREDENTIALS=")[1])
	c = "kubectl --kubeconfig " + k + " -n " + b.csiNamespace + " create secret generic cloud-sa --from-literal=cloud-sa.json='" + string(secret) + "'"
	err = executeCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create CSI secret in CSI namespace")
	}

	// Apply CSI manifest
	in := strings.NewReader(csiManifest)
	cmd := n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
	cmd.SetStdin(in)

	return cmd.Run()
}
