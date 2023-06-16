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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v3"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

//go:embed files/azure/azure-storage-classes.yaml
var azureStorageClasses string

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
	b.capxVersion = "v1.9.2"
	b.capxImageVersion = "v1.9.2"
	b.capxName = "capz"
	b.stClassName = "default"
	b.csiNamespace = "kube-system"
	if managed {
		b.capxTemplate = "azure.aks.tmpl"
	} else {
		b.capxTemplate = "azure.tmpl"
	}
}

func (b *AzureBuilder) setCapxEnvVars(p commons.ProviderParams) {
	b.capxEnvVars = []string{
		"AZURE_CLIENT_SECRET=" + p.Credentials["ClientSecret"],
		"EXP_MACHINE_POOL=true",
	}
	if p.GithubToken != "" {
		b.capxEnvVars = append(b.capxEnvVars, "GITHUB_TOKEN="+p.GithubToken)
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
	var cmd exec.Cmd
	var err error

	c = "helm install azuredisk-csi-driver /stratio/helm/azuredisk-csi-driver" +
		" --kubeconfig " + k
	err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Azure Disk CSI driver Helm Chart")
	}

	cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
	if err = cmd.SetStdin(strings.NewReader(azureStorageClasses)).Run(); err != nil {
		return errors.Wrap(err, "failed to create Azure Storage Classes")
	}

	return nil
}

func (b *AzureBuilder) getAzs(networks commons.Networks) ([]string, error) {
	return []string{"1", "2", "3"}, nil
}

func installCloudProvider(n nodes.Node, descriptorFile commons.DescriptorFile, k string, clusterName string) error {
	var c string
	var err error
	var podsCidrBlock string

	if descriptorFile.Networks.PodsCidrBlock != "" {
		podsCidrBlock = descriptorFile.Networks.PodsCidrBlock
	} else {
		podsCidrBlock = "192.168.0.0/16"
	}

	c = "helm install cloud-provider-azure /stratio/helm/cloud-provider-azure" +
		" --kubeconfig " + k +
		" --set infra.clusterName=" + clusterName +
		" --set 'cloudControllerManager.clusterCIDR=" + podsCidrBlock + "'"
	err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy cloud-provider-azure Helm Chart")
	}

	return nil
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

func getAcrToken(p commons.ProviderParams, acrService string) (string, error) {
	creds, err := azidentity.NewClientSecretCredential(
		p.Credentials["TenantID"], p.Credentials["ClientID"], p.Credentials["ClientSecret"], nil,
	)
	if err != nil {
		return "", err
	}
	ctx := context.Background()

	aadToken, err := creds.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"https://management.azure.com/.default"}})
	if err != nil {
		return "", err
	}
	formData := url.Values{
		"grant_type":   {"access_token"},
		"service":      {acrService},
		"tenant":       {p.Credentials["TenantID"]},
		"access_token": {aadToken.Token},
	}
	jsonResponse, err := http.PostForm(fmt.Sprintf("https://%s/oauth2/exchange", acrService), formData)
	if err != nil {
		return "", err
	}
	var response map[string]interface{}
	json.NewDecoder(jsonResponse.Body).Decode(&response)
	return response["refresh_token"].(string), nil
}
