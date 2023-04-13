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
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
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
	b.capxVersion = "v1.8.2"
	b.capxImageVersion = "v1.8.2"
	b.capxName = "capz"
	b.stClassName = "managed-csi"
	b.csiNamespace = "kube-system"
	if managed {
		b.capxTemplate = "azure.aks.tmpl"
	} else {
		b.capxTemplate = "azure.tmpl"
	}
}

func (b *AzureBuilder) setCapxEnvVars(p ProviderParams) {
	b.capxEnvVars = []string{
		"AZURE_CLIENT_SECRET=" + p.credentials["ClientSecret"],
		"GITHUB_TOKEN=" + p.githubToken,
		"EXP_MACHINE_POOL=true",
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
	var c string
	var err error

	c = "helm install azuredisk-csi-driver /stratio/helm/azuredisk-csi-driver" +
		" --kubeconfig " + k +
		" --namespace " + b.csiNamespace
	err = executeCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Azure Disk CSI driver Helm Chart")
	}

	return nil
}
