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
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

//go:embed files/azure/azure-storage-classes.yaml
var azureStorageClasses string

var storageClassAZTemplate = StorageClassDef{
	APIVersion: "storage.k8s.io/v1",
	Kind:       "StorageClass",
	Metadata: struct {
		Annotations map[string]string `yaml:"annotations,omitempty"`
		Name        string            `yaml:"name"`
	}{
		Annotations: map[string]string{
			defaultScAnnotation: "true",
		},
		Name: "keos",
	},
	Provisioner:       "disk.csi.azure.com",
	Parameters:        make(map[string]interface{}),
	VolumeBindingMode: "WaitForFirstConsumer",
}

var standardAZParameters = commons.SCParameters{
	SkuName: "StandardSSD_LRS",
}

var premiumAZParameters = commons.SCParameters{
	SkuName: "Premium_LRS",
}

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
	b.capxVersion = "v1.9.3"
	b.capxImageVersion = "v1.9.3"
	b.capxName = "capz"
	b.stClassName = "keos"
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
	var err error
	var cmd exec.Cmd

	c = "helm install azuredisk-csi-driver /stratio/helm/azuredisk-csi-driver " +
		" --kubeconfig " + k +
		" --namespace " + b.csiNamespace
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Azure Disk CSI driver Helm Chart")
	}

	cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
	if err := cmd.SetStdin(strings.NewReader(azureStorageClasses)).Run(); err != nil {
		return errors.Wrap(err, "failed to create Azure Storage Classes")
	}

	return nil
}

func (b *AzureBuilder) setStorageClassParameters(storageClass string, params map[string]string) (string, error) {
	return "", nil
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
	_, err = commons.ExecuteCommand(n, c)
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

func (b *AzureBuilder) configureStorageClass(n nodes.Node, k string, sc commons.StorageClass) error {
	var cmd exec.Cmd

	cmd = n.Command("kubectl", "--kubeconfig", k, "annotate", "sc", "default", defaultScAnnotation+"-")
	if err := cmd.SetStdin(strings.NewReader(azureStorageClasses)).Run(); err != nil {
		return errors.Wrap(err, "failed to unannotate default Azure Storage Classes")
	}

	params := b.getParameters(sc)
	storageClass, err := insertParameters(storageClassAZTemplate, params)
	if err != nil {
		return err
	}

	cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
	if err = cmd.SetStdin(strings.NewReader(storageClass)).Run(); err != nil {
		return errors.Wrap(err, "failed to create StorageClass")
	}
	return nil

}

func (b *AzureBuilder) getParameters(sc commons.StorageClass) commons.SCParameters {
	if sc.EncryptionKey != "" {
		sc.Parameters.DiskEncryptionSetID = sc.EncryptionKey
	}
	switch class := sc.Class; class {
	case "standard":
		return mergeSCParameters(sc.Parameters, standardAZParameters)
	case "premium":
		return mergeSCParameters(sc.Parameters, premiumAZParameters)
	default:
		return mergeSCParameters(sc.Parameters, standardAZParameters)
	}
}

func (b *AzureBuilder) internalNginx(networks commons.Networks, credentialsMap map[string]string, ClusterID string) (bool, error) {
	var resourceGroup string
	os.Setenv("AZURE_CLIENT_ID", credentialsMap["ClientID"])
	os.Setenv("AZURE_SECRET_ID", credentialsMap["ClientSecret"])
	os.Setenv("AZURE_TENANT_ID", credentialsMap["TenantID"])

	creds, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return false, err
	}
	ctx := context.Background()

	networkClientFactory, err := armnetwork.NewClientFactory(credentialsMap["SubscriptionID"], creds, nil)
	if err != nil {
		return false, err
	}

	subnetsClient := networkClientFactory.NewSubnetsClient()

	if networks.Subnets != nil {
		if networks.ResourceGroup != "" {
			resourceGroup = networks.ResourceGroup
		} else {
			resourceGroup = ClusterID
		}
		for _, subnet := range networks.Subnets {
			publicSubnetID, _ := AzureFilterPublicSubnet(ctx, subnetsClient, resourceGroup, networks.VPCID, subnet.SubnetId)
			if len(publicSubnetID) > 0 {
				return false, nil
			}
		}
		return true, nil
	}

	return false, nil
}

func AzureFilterPublicSubnet(ctx context.Context, subnetsClient *armnetwork.SubnetsClient, resourceGroup string, VPCID string, subnetID string) (string, error) {
	subnet, err := subnetsClient.Get(ctx, resourceGroup, VPCID, subnetID, nil)
	if err != nil {
		return "", err
	}

	if subnet.Properties.NatGateway != nil && strings.Contains(*subnet.Properties.NatGateway.ID, "natGateways") {
		return "", nil
	} else {
		return subnetID, nil
	}
}
