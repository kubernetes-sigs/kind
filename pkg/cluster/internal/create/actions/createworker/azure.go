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
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

//go:embed files/azure/azure-storage-classes.yaml
var azureStorageClasses string

//go:embed files/azure/internal-ingress-nginx.yaml
var azureInternalIngress []byte

type AzureBuilder struct {
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

func newAzureBuilder() *AzureBuilder {
	return &AzureBuilder{}
}

func (b *AzureBuilder) setCapx(managed bool) {
	b.capxProvider = "azure"
	b.capxVersion = "v1.10.0"
	b.capxImageVersion = "v1.10.0"
	b.capxName = "capz"

	b.csiNamespace = "kube-system"

	if managed {
		b.capxManaged = true
		b.capxTemplate = "azure.aks.tmpl"
	} else {
		b.capxManaged = false
		b.capxTemplate = "azure.tmpl"
	}
}

func (b *AzureBuilder) setSC(p ProviderParams) {
	if (p.StorageClass.Parameters != commons.SCParameters{}) {
		b.scParameters = p.StorageClass.Parameters
	}

	if b.scParameters.Provisioner == "" {
		b.scProvisioner = "disk.csi.azure.com"
	} else {
		b.scProvisioner = b.scParameters.Provisioner
		b.scParameters.Provisioner = ""
	}

	if b.scParameters.SkuName == "" {
		if p.StorageClass.Class == "premium" {
			b.scParameters.SkuName = "Premium_LRS"
		} else {
			b.scParameters.SkuName = "StandardSSD_LRS"
		}
	}

	if p.StorageClass.EncryptionKey != "" {
		b.scParameters.DiskEncryptionSetID = p.StorageClass.EncryptionKey
	}
}

func (b *AzureBuilder) setCapxEnvVars(p ProviderParams) {
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
		capxManaged:      b.capxManaged,
		capxName:         b.capxName,
		capxTemplate:     b.capxTemplate,
		capxEnvVars:      b.capxEnvVars,
		scParameters:     b.scParameters,
		scProvisioner:    b.scProvisioner,
		csiNamespace:     b.csiNamespace,
	}
}

func (b *AzureBuilder) installCSI(n nodes.Node, k string) error {
	var c string
	var err error

	// Deploy CSI driver
	c = "helm install azuredisk-csi-driver /stratio/helm/azuredisk-csi-driver " +
		" --kubeconfig " + k +
		" --namespace " + b.csiNamespace
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Azure Disk CSI driver Helm Chart")
	}

	return nil
}

func (b *AzureBuilder) getAzs(networks commons.Networks) ([]string, error) {
	return []string{"1", "2", "3"}, nil
}

func installCloudProvider(n nodes.Node, keosCluster commons.KeosCluster, k string, clusterName string) error {
	var c string
	var err error
	var podsCidrBlock string

	if keosCluster.Spec.Networks.PodsCidrBlock != "" {
		podsCidrBlock = keosCluster.Spec.Networks.PodsCidrBlock
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

func getAcrToken(p ProviderParams, acrService string) (string, error) {
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

func (b *AzureBuilder) configureStorageClass(n nodes.Node, k string) error {
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

	if !b.capxManaged {
		// Create Azure storage classes
		cmd = n.Command("kubectl", "--kubeconfig", k, "apply", "-f", "-")
		if err := cmd.SetStdin(strings.NewReader(azureStorageClasses)).Run(); err != nil {
			return errors.Wrap(err, "failed to create Azure storage classes")
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

func (b *AzureBuilder) internalNginx(networks commons.Networks, credentialsMap map[string]string, clusterName string) (bool, error) {
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
			resourceGroup = clusterName
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

func (b *AzureBuilder) getOverrideVars(keosCluster commons.KeosCluster, credentialsMap map[string]string) (map[string][]byte, error) {
	overrideVars := map[string][]byte{}
	InternalNginxOVPath, InternalNginxOVValue, err := b.getInternalNginxOverrideVars(keosCluster.Spec.Networks, credentialsMap, keosCluster.Metadata.Name)
	if err != nil {
		return nil, err
	}
	overrideVars = addOverrideVar(InternalNginxOVPath, InternalNginxOVValue, overrideVars)
	return overrideVars, nil
}

func (b *AzureBuilder) getInternalNginxOverrideVars(networks commons.Networks, credentialsMap map[string]string, ClusterID string) (string, []byte, error) {
	requiredInternalNginx, err := b.internalNginx(networks, credentialsMap, ClusterID)
	if err != nil {
		return "", nil, err
	}

	if requiredInternalNginx {
		return "ingress-nginx.yaml", azureInternalIngress, nil
	}

	return "", []byte(""), nil
}
