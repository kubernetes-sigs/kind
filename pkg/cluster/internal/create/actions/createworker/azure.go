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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
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
	b.capxVersion = "v1.8.2"
	b.capxImageVersion = "v1.8.2"
	b.capxName = "capz"
	b.stClassName = "default"
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
	return nil
}

func (b *AzureBuilder) getAzs() ([]string, error) {
	return []string{"1", "2", "3"}, nil
}

func assignUserIdentity(i string, c string, r string, s map[string]string) error {
	creds, err := azidentity.NewClientSecretCredential(s["TenantID"], s["ClientID"], s["ClientSecret"], nil)
	if err != nil {
		return err
	}
	ctx := context.Background()

	containerserviceClientFactory, err := armcontainerservice.NewClientFactory(s["SubscriptionID"], creds, nil)
	if err != nil {
		return err
	}
	managedClustersClient := containerserviceClientFactory.NewManagedClustersClient()

	pollerResp, err := managedClustersClient.BeginCreateOrUpdate(
		ctx, c, c,
		armcontainerservice.ManagedCluster{
			Location: to.Ptr(r),
			Identity: &armcontainerservice.ManagedClusterIdentity{
				Type: to.Ptr(armcontainerservice.ResourceIdentityTypeUserAssigned),
				UserAssignedIdentities: map[string]*armcontainerservice.ManagedServiceIdentityUserAssignedIdentitiesValue{
					i: {},
				},
			},
			Properties: &armcontainerservice.ManagedClusterProperties{
				IdentityProfile: map[string]*armcontainerservice.UserAssignedIdentity{
					"kubeletidentity": {
						ResourceID: to.Ptr(i),
					},
				},
			},
		},
		nil,
	)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
