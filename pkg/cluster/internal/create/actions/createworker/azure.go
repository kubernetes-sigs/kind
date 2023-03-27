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
	b64 "encoding/base64"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
)

type AzureBuilder struct {
	capxProvider     string
	capxVersion      string
	capxImageVersion string
	capxName         string
	capxTemplate     string
	capxEnvVars      []string
	stClassName      string
	csiNamespace     string
}

func newAzureBuilder() *AzureBuilder {
	return &AzureBuilder{}
}

func (b *AzureBuilder) setCapx(managed bool) {
	b.capxProvider = "azure"
	b.capxVersion = "v1.8.1"
	b.capxImageVersion = "v1.8.1"
	b.capxName = "capz"
	b.stClassName = "managed-csi"
	if managed {
		b.capxTemplate = "azure.aks.tmpl"
		b.csiNamespace = ""
	} else {
		b.capxTemplate = "azure.tmpl"
		b.csiNamespace = ""
	}
}

func (b *AzureBuilder) setCapxEnvVars(p ProviderParams) {
	b.capxEnvVars = []string{
		"AZURE_SUBSCRIPTION_ID_B64=" + b64.StdEncoding.EncodeToString([]byte(p.credentials["SubscriptionID"])),
		"AZURE_TENANT_ID_B64=" + b64.StdEncoding.EncodeToString([]byte(p.credentials["TenantID"])),
		"AZURE_CLIENT_SECRET_B64=" + b64.StdEncoding.EncodeToString([]byte(p.credentials["ClientSecret"])),
		"AZURE_CLIENT_ID_B64=" + b64.StdEncoding.EncodeToString([]byte(p.credentials["ClientID"])),
		"GITHUB_TOKEN=" + p.githubToken,
	}
}

func (b *AzureBuilder) getProvider() Provider {
	return Provider{
		capxProvider:     b.capxProvider,
		capxVersion:      b.capxVersion,
		capxImageVersion: b.capxImageVersion,
		capxName:         b.capxName,
		capxTemplate:     b.capxTemplate,
		capxEnvVars:      b.capxEnvVars,
		stClassName:      b.stClassName,
		csiNamespace:     b.csiNamespace,
	}
}

func (b *AzureBuilder) installCSI(n nodes.Node, k string) error {
	return nil
}
