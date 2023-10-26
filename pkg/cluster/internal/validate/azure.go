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

package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
)

const (
	AKSMaxNodeNameLength = 9
)

var AzureVolumes = []string{"Standard_LRS", "Premium_LRS", "StandardSSD_LRS", "UltraSSD_LRS", "Premium_ZRS", "StandardSSD_ZRS", "PremiumV2_LRS"}
var AzureAKSVolumes = []string{"Managed", "Ephemeral"}
var isAzureNodeImage = regexp.MustCompile(`(?i)^\/subscriptions\/[\w-]+\/resourceGroups\/[\w\.-]+\/providers\/Microsoft\.Compute\/images\/[\w\.-]+$`).MatchString
var AzureNodeImageFormat = "/subscriptions/[SUBSCRIPTION_ID]/resourceGroups/[RESOURCE_GROUP]/providers/Microsoft.Compute/images/[IMAGE_NAME]"
var isAzureIdentity = regexp.MustCompile(`(?i)^\/subscriptions\/[\w-]+\/resourcegroups\/[\w\.-]+\/providers\/Microsoft\.ManagedIdentity\/userAssignedIdentities\/[\w\.-]+$`).MatchString
var AzureIdentityFormat = "/subscriptions/[SUBSCRIPTION_ID]/resourceGroups/[RESOURCE_GROUP]/providers/Microsoft.ManagedIdentity/userAssignedIdentities/[IDENTITY_NAME]"
var isPremium = regexp.MustCompile(`^(Premium|Ultra).*$`).MatchString

func validateAzure(spec commons.Spec, providerSecrets map[string]string, clusterName string) error {
	var err error

	creds, err := validateAzureCredentials(providerSecrets)
	if err != nil {
		return err
	}

	regions, err := getAzureRegions(creds, providerSecrets["SubscriptionID"])
	if err != nil {
		return err
	}
	if !commons.Contains(regions, spec.Region) {
		return errors.New("spec.region: " + spec.Region + " region does not exist")
	}

	azs, err := getAzureAzs(creds, providerSecrets["SubscriptionID"], spec.Region)
	if err != nil {
		return err
	}

	for _, wn := range spec.WorkerNodes {
		if wn.AZ != "" {
			if len(azs) > 0 {
				if !commons.Contains(azs, wn.AZ) {
					return errors.New(wn.AZ + " does not exist in this region, azs: " + fmt.Sprint(azs))
				}
			}
		}
	}

	if (spec.StorageClass != commons.StorageClass{}) {
		if err = validateAzureStorageClass(spec.StorageClass, spec.WorkerNodes); err != nil {
			return errors.Wrap(err, "spec.storageclass: Invalid value")
		}
	}
	if !reflect.ValueOf(spec.Networks).IsZero() {
		if err = validateAzureNetwork(spec.Networks, spec, creds, providerSecrets["SubscriptionID"], clusterName); err != nil {
			return errors.Wrap(err, "spec.networks: Invalid value")
		}
	}
	if !isAzureIdentity(spec.Security.ControlPlaneIdentity) {
		return errors.New("spec.security: Invalid value: \"control_plane_identity\": is required and have the format " + AzureIdentityFormat)
	}
	if spec.Security.NodesIdentity != "" {
		if !isAzureIdentity(spec.Security.NodesIdentity) {
			return errors.New("spec.security: Invalid value: \"nodes_identity\": it must have the format " + AzureIdentityFormat)
		}
	}

	for i, dr := range spec.DockerRegistries {
		if dr.Type != "acr" && spec.ControlPlane.Managed {
			return errors.New("spec.docker_registries[" + strconv.Itoa(i) + "]: Invalid value: \"type\": only acr is supported in azure managed clusters")
		}
		if dr.Type != "acr" && dr.Type != "generic" {
			return errors.New("spec.docker_registries[" + strconv.Itoa(i) + "]: Invalid value: \"type\": only acr and generic are supported in azure unmanaged clusters")
		}
	}

	if spec.ControlPlane.Managed {
		if err = validateAKSVersion(spec, creds, providerSecrets["SubscriptionID"]); err != nil {
			return err
		}
		if err = validateAKSNodes(spec.WorkerNodes); err != nil {
			return err
		}
	}

	if !spec.ControlPlane.Managed {
		if spec.ControlPlane.NodeImage != "" {
			if !isAzureNodeImage(spec.ControlPlane.NodeImage) {
				return errors.New("spec.control_plane: Invalid value: \"node_image\": must have the format " + AzureNodeImageFormat)
			}
		}
		if err := validateVolumeType(spec.ControlPlane.RootVolume.Type, AzureVolumes); err != nil {
			return errors.Wrap(err, "spec.control_plane.root_volume: Invalid value: \"type\"")
		}
		for i, ev := range spec.ControlPlane.ExtraVolumes {
			if ev.Name == "" {
				return errors.New("spec.control_plane.extra_volumes[" + strconv.Itoa(i) + "]: Required value: \"name\"")
			}
			if err := validateVolumeType(ev.Type, AzureVolumes); err != nil {
				return errors.Wrap(err, "spec.control_plane.extra_volumes["+strconv.Itoa(i)+"]: Invalid value: \"type\"")
			}
			for _, ev2 := range spec.ControlPlane.ExtraVolumes[i+1:] {
				if ev.Name == ev2.Name {
					return errors.New("spec.control_plane.extra_volumes[" + strconv.Itoa(i) + "]: Invalid value: \"name\": is duplicated")
				}
			}
		}
		for _, wn := range spec.WorkerNodes {
			if wn.NodeImage != "" {
				if !isAzureNodeImage(wn.NodeImage) {
					return errors.New("spec.worker_nodes." + wn.Name + ": \"node_image\": must have the format " + AzureNodeImageFormat)
				}
			}
			if err := validateVolumeType(wn.RootVolume.Type, AzureVolumes); err != nil {
				return errors.Wrap(err, "spec.worker_nodes."+wn.Name+".root_volume: Invalid value: \"type\"")
			}

			premiumStorage := hasAzurePremiumStorage(wn.Size)
			if isPremium(wn.RootVolume.Type) && !premiumStorage {
				return errors.New("spec.worker_nodes." + wn.Name + ".root_volume: Invalid value: \"type\": size doesn't support premium storage")
			}
			for i, ev := range wn.ExtraVolumes {
				if ev.Name == "" {
					return errors.New("spec.worker_nodes." + wn.Name + ".extra_volumes[" + strconv.Itoa(i) + "]: Required value: \"name\"")
				}
				if err := validateVolumeType(ev.Type, AzureVolumes); err != nil {
					return errors.Wrap(err, "spec.worker_nodes."+wn.Name+".extra_volumes["+strconv.Itoa(i)+"]: Invalid value: \"type\"")
				}
				if isPremium(ev.Type) && !premiumStorage {
					return errors.New("spec.worker_nodes." + wn.Name + ".extra_volumes[" + strconv.Itoa(i) + "]: Invalid value: \"type\": size doesn't support premium storage")
				}
				for _, ev2 := range wn.ExtraVolumes[i+1:] {
					if ev.Name == ev2.Name {
						return errors.New("spec.worker_nodes." + wn.Name + ".extra_volumes[" + strconv.Itoa(i) + "]: Invalid value: \"name\": is duplicated")
					}
				}
			}
		}
	}

	return nil
}

func validateAzureCredentials(secrets map[string]string) (*azidentity.ClientSecretCredential, error) {
	creds, err := azidentity.NewClientSecretCredential(secrets["TenantID"], secrets["ClientID"], secrets["ClientSecret"], nil)
	if err != nil {
		return &azidentity.ClientSecretCredential{}, err
	}
	return creds, nil
}

func validateAzureStorageClass(sc commons.StorageClass, wn commons.WorkerNodes) error {
	var err error
	var isKeyValid = regexp.MustCompile(`(?i)^\/subscriptions\/[\w-]+\/resourceGroups\/[\w\.-]+\/providers\/Microsoft\.Compute\/diskEncryptionSets\/[\w\.-]+$`).MatchString
	var AzureFSTypes = []string{"xfs", "ext3", "ext4", "ext2", "btrfs"}
	var AzureSCFields = []string{"FsType", "Kind", "CachingMode", "DiskAccessID", "DiskEncryptionSetID", "DiskEncryptionType", "EnableBursting", "EnablePerformancePlus", "NetworkAccessPolicy", "Provisioner", "PublicNetworkAccess", "ResourceGroup", "SkuName", "SubscriptionID", "Tags"}
	var AzureSCYamlFields = []string{"fsType", "kind", "cachingMode", "diskAccessID", "diskEncryptionSetID", "diskEncryptionType", "enableBursting", "enablePerformancePlus", "networkAccessPolicy", "provisioner", "publicNetworkAccess", "resourceGroup", "skuName", "subscriptionID", "tags"}

	// Validate fields
	fields := getFieldNames(sc.Parameters)
	for _, f := range fields {
		if !commons.Contains(AzureSCFields, f) {
			return errors.New("\"parameters\": unsupported " + f + ", supported fields: " + fmt.Sprint(strings.Join(AzureSCYamlFields, ", ")))
		}
	}
	// Validate class
	if sc.Class != "" && sc.Parameters != (commons.SCParameters{}) {
		return errors.New("\"class\": cannot be set when \"parameters\" is set")
	}
	// Validate type
	if sc.Parameters.SkuName != "" && !commons.Contains(AzureVolumes, sc.Parameters.SkuName) {
		return errors.New("\"type\": unsupported " + sc.Parameters.Type + ", supported types: " + fmt.Sprint(strings.Join(AzureVolumes, ", ")))
	}
	// Validate encryptionKey format
	if sc.EncryptionKey != "" {
		if sc.Parameters != (commons.SCParameters{}) {
			return errors.New("\"encryptionKey\": cannot be set when \"parameters\" is set")
		}
		if !isKeyValid(sc.EncryptionKey) {
			return errors.New("\"encryptionKey\": it must have the format /subscriptions/[SUBSCRIPTION_ID]/resourceGroups/[RESOURCE_GROUP]/providers/Microsoft.ManagedIdentity/diskEncryptionSets/[DISK_ENCRYPION_SETS_NAME]")
		}
	}
	// Validate diskEncryptionSetID format
	if sc.Parameters.DiskEncryptionSetID != "" {
		if !isKeyValid(sc.Parameters.DiskEncryptionSetID) {
			return errors.New("incorrect diskEncryptionSetID format. It must have the format /subscriptions/[SUBSCRIPTION_ID]/resourceGroups/[RESOURCE_GROUP]/providers/Microsoft.ManagedIdentity/diskEncryptionSets/[DISK_ENCRYPION_SETS_NAME]")
		}
	}
	// Validate fsType
	if sc.Parameters.FsType != "" && !commons.Contains(AzureFSTypes, sc.Parameters.FsType) {
		return errors.New("unsupported fsType: " + sc.Parameters.FsType + ". Supported types: " + fmt.Sprint(strings.Join(AzureFSTypes, ", ")))
	}
	// Validate size support premium storage
	if sc.Class == "premium" || isPremium(sc.Parameters.SkuName) {
		hasPremium := false
		for _, n := range wn {
			if hasAzurePremiumStorage(n.Size) {
				hasPremium = true
				break
			}
		}
		if !hasPremium {
			return errors.New("premium storage is not supported in any workers nodes")
		}
	}
	// Validate cachingMode
	if sc.Parameters.CachingMode == "ReadOnly" && sc.Parameters.SkuName == "PremiumV2_LRS" {
		return errors.New("with skuName PremiumV2_LRS, CachingMode only can be none")
	}
	// Validate tags
	if sc.Parameters.Tags != "" {
		if err = validateLabel(sc.Parameters.Tags); err != nil {
			return errors.Wrap(err, "invalid tags")
		}
	}
	return nil
}

func validateAzureNetwork(network commons.Networks, spec commons.Spec, creds *azidentity.ClientSecretCredential, subscription string, clusterName string) error {
	rg := clusterName
	if network.VPCID != "" {

		if spec.Networks.ResourceGroup != "" {
			rg = spec.Networks.ResourceGroup
		}
		vpcs, err := getAzureVpcs(creds, subscription, spec.Region, rg)
		if err != nil {
			return err
		}
		if len(vpcs) > 0 && !commons.Contains(vpcs, network.VPCID) {
			return errors.New("\"vpc_id\": " + network.VPCID + " does not exist in this resourceGroup")
		}
		if len(network.Subnets) == 0 {
			return errors.New("\"subnets\": are required when \"vpc_id\" is set")
		}
		if spec.ControlPlane.Managed && network.VPCCidrBlock == "" {
			return errors.New("\"vpc_cidr\": is required when \"vpc_id\" is set")
		}
	} else {
		if len(network.Subnets) > 0 {
			return errors.New("\"vpc_id\": is required when \"subnets\" is set")
		}
		if network.VPCCidrBlock != "" {
			if spec.ControlPlane.Managed {
				return errors.New("\"vpc_id\": is required when \"vpc_cidr\" is set")
			} else {
				return errors.New("\"vpc_cidr\": is only supported in azure managed clusters")
			}
		}
	}
	if len(network.Subnets) > 0 {
		subnets, err := getAzureSubnets(creds, subscription, rg, network.VPCID)
		if err != nil {
			return err
		}
		for _, s := range network.Subnets {
			if s.SubnetId == "" {
				return errors.New("\"subnet_id\": is required")
			}
			if len(subnets) > 0 && !commons.Contains(subnets, s.SubnetId) {
				return errors.New("\"subnet_id\": " + s.SubnetId + " does not belong to VPC: " + network.VPCID + " and resourceGroup: " + rg)
			}
			if spec.ControlPlane.Managed {
				if s.CidrBlock == "" {
					return errors.New("\"cidr\": is required")
				}
				if s.Role != "" {
					return errors.New("\"role\": is only supported in azure unmanaged clusters")
				}
			} else {
				if s.Role == "" {
					return errors.New("\"role\": is required")
				}
				if s.CidrBlock != "" {
					return errors.New("\"cidr\": is only supported in azure managed clusters")
				}
			}
		}
	}
	return nil
}

func validateAKSVersion(spec commons.Spec, creds *azidentity.ClientSecretCredential, subscription string) error {
	var availableVersions []string
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(subscription, creds, nil)
	if err != nil {
		return err
	}
	res, err := clientFactory.NewManagedClustersClient().ListKubernetesVersions(ctx, spec.Region, nil)
	if err != nil {
		return err
	}
	for _, v := range res.KubernetesVersionListResult.Values {
		for _, p := range v.PatchVersions {
			for _, u := range p.Upgrades {
				if !commons.Contains(availableVersions, *u) {
					availableVersions = append(availableVersions, *u)
				}
			}
		}
	}
	if !slices.Contains(availableVersions, strings.ReplaceAll(spec.K8SVersion, "v", "")) {
		a, _ := json.Marshal(availableVersions)
		return errors.New("spec: Invalid value: \"k8s_version\": AKS only supports Kubernetes versions: " + string(a))
	}
	return nil
}

func validateAKSNodes(wn commons.WorkerNodes) error {
	var isLetter = regexp.MustCompile(`^[a-z0-9]+$`).MatchString
	for _, n := range wn {
		if !isLetter(n.Name) || len(n.Name) >= AKSMaxNodeNameLength {
			return errors.New("spec.worker_nodes." + n.Name + " : Invalid value \"name\": in AKS must be " + strconv.Itoa(AKSMaxNodeNameLength) + " characters or less & contain only lowercase alphanumeric characters")
		}
		if n.RootVolume.Type != "" && !commons.Contains(AzureAKSVolumes, n.RootVolume.Type) {
			return errors.New("spec.worker_nodes." + n.Name + ".root_volume: Invalid value \"type\": " + n.RootVolume.Type + " unsupported, supported types: " + fmt.Sprint(strings.Join(AzureAKSVolumes, ", ")))
		}
	}
	return nil
}

func hasAzurePremiumStorage(s string) bool {
	return strings.Contains(strings.ToLower(strings.ReplaceAll(s, "Standard_", "")), "s")
}

func getAzureAzs(creds *azidentity.ClientSecretCredential, subscription string, region string) ([]string, error) {
	azs := []string{}

	ctx := context.Background()
	clientFactory, err := armsubscriptions.NewClientFactory(creds, nil)
	if err != nil {
		return []string{}, err
	}
	pager := clientFactory.NewClient().NewListLocationsPager(subscription, &armsubscriptions.ClientListLocationsOptions{IncludeExtendedLocations: nil})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return []string{}, err
		}
		for _, v := range page.Value {
			if *v.Name == region {
				for _, az := range v.AvailabilityZoneMappings {
					azs = append(azs, *az.LogicalZone)
				}
				break
			}
		}
	}

	return azs, nil
}

func getAzureRegions(creds *azidentity.ClientSecretCredential, subscription string) ([]string, error) {
	regions := []string{}

	ctx := context.Background()
	clientFactory, err := armsubscriptions.NewClientFactory(creds, nil)
	if err != nil {
		return []string{}, err
	}

	pager := clientFactory.NewClient().NewListLocationsPager(subscription, &armsubscriptions.ClientListLocationsOptions{IncludeExtendedLocations: nil})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return []string{}, err
		}
		for _, v := range page.Value {
			if !commons.Contains(regions, *v.Name) {
				regions = append(regions, *v.Name)
			}
		}
	}
	return regions, nil
}

func getAzureVpcs(creds *azidentity.ClientSecretCredential, subscription string, region string, resourceGroup string) ([]string, error) {
	ctx := context.Background()
	vpcs := []string{}
	clientFactory, err := armnetwork.NewClientFactory(subscription, creds, nil)
	if err != nil {
		return []string{}, err
	}
	pager := clientFactory.NewVirtualNetworksClient().NewListPager(resourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return []string{}, err
		}
		for _, v := range page.Value {
			if *v.Location == region {
				vpcs = append(vpcs, *v.Name)
			}
		}
	}
	return vpcs, nil
}

func getAzureSubnets(creds *azidentity.ClientSecretCredential, subscription string, resourceGroup string, vpcId string) ([]string, error) {
	ctx := context.Background()
	subnets := []string{}
	clientFactory, err := armnetwork.NewClientFactory(subscription, creds, nil)
	if err != nil {
		return []string{}, err
	}
	pager := clientFactory.NewSubnetsClient().NewListPager(resourceGroup, vpcId, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return []string{}, err
		}
		for _, v := range page.Value {
			subnets = append(subnets, *v.Name)
		}
	}
	return subnets, nil
}
