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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
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
	b.capxVersion = "v1.10.4"
	b.capxImageVersion = "v1.10.4"
	b.capxName = "capz"
	b.capxManaged = managed
	b.csiNamespace = "kube-system"
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
		"AZURE_SUBSCRIPTION_ID_B64=" + base64.StdEncoding.EncodeToString([]byte(p.Credentials["SubscriptionID"])),
		"AZURE_TENANT_ID_B64=" + base64.StdEncoding.EncodeToString([]byte(p.Credentials["TenantID"])),
		"AZURE_CLIENT_ID_B64=" + base64.StdEncoding.EncodeToString([]byte(p.Credentials["ClientID"])),
		"AZURE_CLIENT_SECRET_B64=" + base64.StdEncoding.EncodeToString([]byte(p.Credentials["ClientSecret"])),
	}
	if p.Managed {
		b.capxEnvVars = append(b.capxEnvVars, "EXP_MACHINE_POOL=true")
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
		capxEnvVars:      b.capxEnvVars,
		scParameters:     b.scParameters,
		scProvisioner:    b.scProvisioner,
		csiNamespace:     b.csiNamespace,
	}
}

func (b *AzureBuilder) installCloudProvider(n nodes.Node, k string, keosCluster commons.KeosCluster) error {
	var podsCidrBlock string
	if keosCluster.Spec.Networks.PodsCidrBlock != "" {
		podsCidrBlock = keosCluster.Spec.Networks.PodsCidrBlock
	} else {
		podsCidrBlock = "192.168.0.0/16"
	}
	c := "helm install cloud-provider-azure /stratio/helm/cloud-provider-azure" +
		" --kubeconfig " + k +
		" --namespace kube-system" +
		" --set infra.clusterName=" + keosCluster.Metadata.Name +
		" --set 'cloudControllerManager.clusterCIDR=" + podsCidrBlock + "'"
	_, err := commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy cloud-provider-azure Helm Chart")
	}
	return nil
}

func (b *AzureBuilder) installCSI(n nodes.Node, k string) error {
	var c string
	var err error

	// Deploy disk CSI driver
	c = "helm install azuredisk-csi-driver /stratio/helm/azuredisk-csi-driver " +
		" --kubeconfig " + k +
		" --namespace " + b.csiNamespace
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Azure Disk CSI driver Helm Chart")
	}

	// Deploy file CSI driver
	c = "helm install azurefile-csi-driver /stratio/helm/azurefile-csi-driver " +
		" --kubeconfig " + k +
		" --namespace " + b.csiNamespace
	_, err = commons.ExecuteCommand(n, c)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Azure File CSI driver Helm Chart")
	}

	return nil
}

func (b *AzureBuilder) getRegistryCredentials(p ProviderParams, u string) (string, string, error) {
	var registryUser = "00000000-0000-0000-0000-000000000000"
	var registryPass string
	var ctx = context.Background()
	var response map[string]interface{}

	cfg, err := commons.AzureGetConfig(p.Credentials)
	if err != nil {
		return "", "", err
	}
	aadToken, err := cfg.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"https://management.azure.com/.default"}})
	if err != nil {
		return "", "", err
	}
	acrService := strings.Split(u, "/")[0]
	formData := url.Values{
		"grant_type":   {"access_token"},
		"service":      {acrService},
		"tenant":       {p.Credentials["TenantID"]},
		"access_token": {aadToken.Token},
	}
	jsonResponse, err := http.PostForm(fmt.Sprintf("https://%s/oauth2/exchange", acrService), formData)
	if err != nil {
		return "", "", err
	} else if jsonResponse.StatusCode == http.StatusUnauthorized {
		return "", "", errors.New("Failed to obtain the ACR token with the provided credentials, please check the roles assigned to the correspondent Azure AD app")
	}
	json.NewDecoder(jsonResponse.Body).Decode(&response)
	if response["access_token"] != nil {
		registryPass = response["access_token"].(string)
	} else if response["refresh_token"] != nil {
		registryPass = response["refresh_token"].(string)
	} else {
		return "", "", errors.New("Failed to obtain the ACR token with the provided credentials, please check the roles assigned to the correspondent Azure AD app")
	}
	return registryUser, registryPass, nil
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

func (b *AzureBuilder) internalNginx(p ProviderParams, networks commons.Networks) (bool, error) {
	var resourceGroup string
	var ctx = context.Background()

	cfg, err := commons.AzureGetConfig(p.Credentials)
	if err != nil {
		return false, err
	}
	networkClientFactory, err := armnetwork.NewClientFactory(p.Credentials["SubscriptionID"], cfg, nil)
	if err != nil {
		return false, err
	}
	subnetsClient := networkClientFactory.NewSubnetsClient()
	if len(networks.Subnets) > 0 {
		if networks.ResourceGroup != "" {
			resourceGroup = networks.ResourceGroup
		} else {
			resourceGroup = p.ClusterName
		}
		for _, s := range networks.Subnets {
			publicSubnetID, err := AzureFilterPublicSubnet(ctx, subnetsClient, resourceGroup, networks.VPCID, s.SubnetId)
			if err != nil || len(publicSubnetID) > 0 {
				return false, err
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

func (b *AzureBuilder) getOverrideVars(p ProviderParams, networks commons.Networks) (map[string][]byte, error) {
	var overrideVars = make(map[string][]byte)

	requiredInternalNginx, err := b.internalNginx(p, networks)
	if err != nil {
		return nil, err
	}
	if requiredInternalNginx {
		overrideVars = addOverrideVar("ingress-nginx.yaml", azureInternalIngress, overrideVars)
	}
	return overrideVars, nil
}
