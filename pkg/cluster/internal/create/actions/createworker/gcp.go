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
	"context"
	_ "embed"
	b64 "encoding/base64"
	"encoding/json"
	"net/url"
	"strings"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

//go:embed files/gcp/gcp-compute-persistent-disk-csi-driver.yaml
var csiManifest string

//go:embed files/gcp/internal-ingress-nginx.yaml
var gcpInternalIngress []byte

type GCPBuilder struct {
	capxProvider     string
	capxVersion      string
	capxImageVersion string
	capxManaged      bool
	capxName         string
	capxTemplate     string
	capxEnvVars      []string
	scParameters     commons.SCParameters
	scProvisioner    string
	csiNamespace     string
}

func newGCPBuilder() *GCPBuilder {
	return &GCPBuilder{}
}

func (b *GCPBuilder) setCapx(managed bool) {
	b.capxProvider = "gcp"
	b.capxVersion = "v1.4.0"
	b.capxImageVersion = "v1.4.0"
	b.capxName = "capg"
	b.capxManaged = managed
	if managed {
		b.capxTemplate = "gcp.gke.tmpl"
		b.csiNamespace = ""
	} else {
		b.capxTemplate = "gcp.tmpl"
		b.csiNamespace = "kube-system"
	}
}

func (b *GCPBuilder) setCapxEnvVars(p ProviderParams) {
	data := map[string]interface{}{
		"type":                        "service_account",
		"project_id":                  p.Credentials["ProjectID"],
		"private_key_id":              p.Credentials["PrivateKeyID"],
		"private_key":                 p.Credentials["PrivateKey"],
		"client_email":                p.Credentials["ClientEmail"],
		"client_id":                   p.Credentials["ClientID"],
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://accounts.google.com/o/oauth2/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/" + url.QueryEscape(p.Credentials["ClientEmail"]),
	}
	jsonData, _ := json.Marshal(data)
	b.capxEnvVars = []string{
		"GCP_B64ENCODED_CREDENTIALS=" + b64.StdEncoding.EncodeToString([]byte(jsonData)),
	}
	if p.GithubToken != "" {
		b.capxEnvVars = append(b.capxEnvVars, "GITHUB_TOKEN="+p.GithubToken)
	}
}

func (b *GCPBuilder) setSC(p ProviderParams) {
	if (p.StorageClass.Parameters != commons.SCParameters{}) {
		b.scParameters = p.StorageClass.Parameters
	}

	b.scProvisioner = "pd.csi.storage.gke.io"

	if b.scParameters.Type == "" {
		if p.StorageClass.Class == "premium" {
			b.scParameters.Type = "pd-ssd"
		} else {
			b.scParameters.Type = "pd-standard"
		}
	}

	if p.StorageClass.EncryptionKey != "" {
		b.scParameters.DiskEncryptionKmsKey = p.StorageClass.EncryptionKey
	}
}

func (b *GCPBuilder) getProvider() Provider {
	return Provider{
		capxProvider:     b.capxProvider,
		capxVersion:      b.capxVersion,
		capxImageVersion: b.capxImageVersion,
		capxName:         b.capxName,
		capxTemplate:     b.capxTemplate,
		capxEnvVars:      b.capxEnvVars,
		scParameters:     b.scParameters,
		scProvisioner:    b.scProvisioner,
		csiNamespace:     b.csiNamespace,
	}
}

func (b *GCPBuilder) installCloudProvider(n nodes.Node, k string, keosCluster commons.KeosCluster) error {
	return nil
}

func (b *GCPBuilder) installCSI(n nodes.Node, k string) error {
	var c string
	var err error
	var cmd exec.Cmd

	// Create CSI secret in CSI namespace
	secret, _ := b64.StdEncoding.DecodeString(strings.Split(b.capxEnvVars[0], "GCP_B64ENCODED_CREDENTIALS=")[1])
	c = "kubectl --kubeconfig " + k + " -n " + b.csiNamespace + " create secret generic cloud-sa --from-literal=cloud-sa.json='" + string(secret) + "'"
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to create CSI secret in CSI namespace")
	}

	// Deploy CSI driver
	cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
	if err = cmd.SetStdin(strings.NewReader(csiManifest)).Run(); err != nil {
		return errors.Wrap(err, "failed to deploy CSI driver")
	}

	return nil
}

func (b *GCPBuilder) getAzs(p ProviderParams, networks commons.Networks) ([]string, error) {
	var azs []string
	var ctx = context.Background()

	secrets, _ := b64.StdEncoding.DecodeString(strings.Split(b.capxEnvVars[0], "GCP_B64ENCODED_CREDENTIALS=")[1])
	cfg := option.WithCredentialsJSON(secrets)
	computeService, err := compute.NewService(ctx, cfg)
	if err != nil {
		return nil, err
	}
	zones, err := computeService.Zones.List(p.Credentials["ProjectID"]).Filter("name=" + p.Region + "*").Do()
	if err != nil {
		return nil, err
	}
	if len(zones.Items) < 3 {
		return nil, errors.New("insufficient availability aones in this region. Must have at least 3")
	}
	for i, zone := range zones.Items {
		if i == 3 {
			break
		}
		azs = append(azs, zone.Name)
	}
	return azs, nil
}

func (b *GCPBuilder) configureStorageClass(n nodes.Node, k string) error {
	var c string
	var err error
	var cmd exec.Cmd

	if b.capxManaged {
		// Remove annotation from default storage class
		c = "kubectl --kubeconfig " + k + ` get sc -o jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}'`
		output, err := commons.ExecuteCommand(n, c)
		if err != nil {
			return errors.Wrap(err, "failed to get default storage class")
		}
		if strings.TrimSpace(output) != "" && strings.TrimSpace(output) != "No resources found" {
			c = "kubectl --kubeconfig " + k + " annotate sc " + strings.TrimSpace(output) + " " + defaultScAnnotation + "-"
			_, err = commons.ExecuteCommand(n, c)
			if err != nil {
				return errors.Wrap(err, "failed to remove annotation from default storage class")
			}
		}
	}

	scTemplate.Parameters = b.scParameters
	scTemplate.Provisioner = b.scProvisioner

	scBytes, err := yaml.Marshal(scTemplate)
	if err != nil {
		return err
	}
	storageClass := strings.Replace(string(scBytes), "fsType", "csi.storage.k8s.io/fstype", -1)

	cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
	if err = cmd.SetStdin(strings.NewReader(storageClass)).Run(); err != nil {
		return errors.Wrap(err, "failed to create default storage class")
	}

	return nil
}

func (b *GCPBuilder) internalNginx(p ProviderParams, networks commons.Networks) (bool, error) {
	var ctx = context.Background()

	secrets, _ := b64.StdEncoding.DecodeString(strings.Split(b.capxEnvVars[0], "GCP_B64ENCODED_CREDENTIALS=")[1])
	cfg := option.WithCredentialsJSON(secrets)
	computeService, err := compute.NewService(ctx, cfg)
	if err != nil {
		return false, err
	}
	if len(networks.Subnets) > 0 {
		for _, s := range networks.Subnets {
			publicSubnetID, _ := GCPFilterPublicSubnet(computeService, p.Credentials["ProjectID"], p.Region, s.SubnetId)
			if len(publicSubnetID) > 0 {
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

func GCPFilterPublicSubnet(computeService *compute.Service, projectID string, region string, subnetID string) (string, error) {
	subnet, err := computeService.Subnetworks.Get(projectID, region, subnetID).Do()
	if err != nil {
		return "", err
	}
	if subnet.PrivateIpGoogleAccess {
		return "", nil
	} else {
		return subnetID, nil
	}
}

func (b *GCPBuilder) getOverrideVars(p ProviderParams, networks commons.Networks) (map[string][]byte, error) {
	var overrideVars map[string][]byte

	requiredInternalNginx, err := b.internalNginx(p, networks)
	if err != nil {
		return nil, err
	}
	if requiredInternalNginx {
		overrideVars = addOverrideVar("ingress-nginx.yaml", gcpInternalIngress, overrideVars)
	}
	return overrideVars, nil
}
