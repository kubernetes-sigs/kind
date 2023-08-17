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
	"github.com/iancoleman/strcase"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
)

const (
	AKSMaxNodeNameLength = 9
)

var AzureVolumes = []string{"Standard_LRS", "Premium_LRS", "StandardSSD_LRS", "UltraSSD_LRS", "Premium_ZRS", "StandardSSD_ZRS", "PremiumV2_LRS"}
var AzureAKSVolumes = []string{"Managed", "Ephemeral"}
var AzureFSTypes = []string{"xfs", "ext3", "ext4", "ext2", "btrfs"}
var AzureSCFields = []string{"FsType", "Kind", "CachingMode", "DiskAccessID", "DiskEncryptionType", "EnableBursting", "EnablePerformancePlus", "NetworkAccessPolicy", "Provisioner", "PublicNetworkAccess", "ResourceGroup", "SkuName", "SubscriptionID", "Tags"}

var isAzureNodeImage = regexp.MustCompile(`(?i)^\/subscriptions\/[\w-]+\/resourceGroups\/[\w\.-]+\/providers\/Microsoft\.Compute\/images\/[\w\.-]+$`).MatchString
var AzureNodeImageFormat = "/subscriptions/[SUBSCRIPTION_ID]/resourceGroups/[RESOURCE_GROUP]/providers/Microsoft.Compute/images/[IMAGE_NAME]"

var isAzureIdentity = regexp.MustCompile(`(?i)^\/subscriptions\/[\w-]+\/resourcegroups\/[\w\.-]+\/providers\/Microsoft\.ManagedIdentity\/userAssignedIdentities\/[\w\.-]+$`).MatchString
var AzureIdentityFormat = "/subscriptions/[SUBSCRIPTION_ID]/resourceGroups/[RESOURCE_GROUP]/providers/Microsoft.ManagedIdentity/userAssignedIdentities/[IDENTITY_NAME]"

var isPremium = regexp.MustCompile(`^(Premium|Ultra).*$`).MatchString

func validateAzure(spec commons.Spec, providerSecrets map[string]string) error {
	var err error

	creds, err := validateAzureCredentials(providerSecrets)
	if err != nil {
		return err
	}

	if (spec.StorageClass != commons.StorageClass{}) {
		if err = validateAzureStorageClass(spec.StorageClass, spec.WorkerNodes); err != nil {
			return errors.Wrap(err, "spec.storageclass: Invalid value")
		}
	}
	if !reflect.ValueOf(spec.Networks).IsZero() {
		if err = validateAzureNetwork(spec.Networks, spec.ControlPlane.Managed); err != nil {
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
		if dr.Type == "ecr" {
			return errors.New("spec.docker_registries[" + strconv.Itoa(i) + "]: Invalid value: \"type\": ecr is not supported in Azure/AKS")
		}
		if dr.Type != "acr" && spec.ControlPlane.Managed {
			return errors.New("spec.docker_registries[" + strconv.Itoa(i) + "]: Invalid value: \"type\": acr is not supported in AKS")
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

	// Validate fields
	fields := getFieldNames(sc.Parameters)
	for _, f := range fields {
		if !commons.Contains(AzureSCFields, f) {
			return errors.New("field " + strcase.ToLowerCamel(f) + " is not supported in storage class")
		}
	}

	// Validate encryptionKey format
	if sc.EncryptionKey != "" {
		if !isKeyValid(sc.EncryptionKey) {
			return errors.New("incorrect encryptionKey format. It must have the format /subscriptions/[SUBSCRIPTION_ID]/resourceGroups/[RESOURCE_GROUP]/providers/Microsoft.ManagedIdentity/diskEncryptionSets/[DISK_ENCRYPION_SETS_NAME]")
		}
	}
	// Validate diskEncryptionSetID format
	if sc.Parameters.DiskEncryptionSetID != "" {
		if !isKeyValid(sc.Parameters.DiskEncryptionSetID) {
			return errors.New("incorrect diskEncryptionSetID format. It must have the format /subscriptions/[SUBSCRIPTION_ID]/resourceGroups/[RESOURCE_GROUP]/providers/Microsoft.ManagedIdentity/diskEncryptionSets/[DISK_ENCRYPION_SETS_NAME]")
		}
	}
	// Validate type
	if sc.Parameters.SkuName != "" && !commons.Contains(AzureVolumes, sc.Parameters.SkuName) {
		return errors.New("unsupported skuname: " + sc.Parameters.SkuName)
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

func validateAzureNetwork(network commons.Networks, managed bool) error {
	if network.VPCID == "" {
		return errors.New("vpc_id is required")
	}
	if managed && network.VPCCidrBlock == "" {
		return errors.New("vpc_cidr is required")
	}
	if len(network.Subnets) > 0 {
		for _, s := range network.Subnets {
			if s.SubnetId == "" {
				return errors.New("subnet_id is required")
			}
			if managed && s.CidrBlock == "" {
				return errors.New("cidr is required")
			}
			if !managed && s.Role == "" {
				return errors.New("role is required")
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
				availableVersions = append(availableVersions, *u)
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
