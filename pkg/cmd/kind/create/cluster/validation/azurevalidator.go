package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v3"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/kind/pkg/commons"
)

var azureInstance *AzureValidator

type AzureValidator struct {
	commonValidator
	managed bool
}

var provisionersTypesAzure = []string{"Standard_LRS", "Premium_LRS", "StandardSSD_LRS", "UltraSSD_LRS", "Premium_ZRS", "StandardSSD_ZRS", "PremiumV2_LRS"}

func newAzureValidator(managed bool) *AzureValidator {
	if azureInstance == nil {
		azureInstance = new(AzureValidator)
	}
	azureInstance.managed = managed
	return azureInstance
}

func (v *AzureValidator) DescriptorFile(descriptorFile commons.DescriptorFile) {
	v.descriptor = descriptorFile
}

func (v *AzureValidator) SecretsFile(secrets commons.SecretsFile) {
	v.secrets = secrets
}

func (v *AzureValidator) Validate(fileType string) error {
	switch fileType {
	case "descriptor":
		err := v.descriptorAzureValidations((*v).descriptor, (*v).secrets, (*v).managed)
		if err != nil {
			return err
		}
	case "secrets":
		err := secretsAzureValidations((*v).secrets)
		if err != nil {
			return err
		}
	default:
		return errors.New("incorrect filetype validation")
	}
	return nil
}

func (v *AzureValidator) CommonsValidations() error {
	err := commonsValidations((*v).descriptor, (*v).secrets)
	if err != nil {
		return err
	}
	return nil
}

func (v *AzureValidator) descriptorAzureValidations(descriptorFile commons.DescriptorFile, secretsFile commons.SecretsFile, managed bool) error {
	err := commonsDescriptorValidation(descriptorFile)
	if err != nil {
		return err
	}
	if managed {
		err = aksVersionValidation(descriptorFile, secretsFile)
		if err != nil {
			return err
		}
		err = aksNodesValidation(descriptorFile.WorkerNodes)
		if err != nil {
			return err
		}
	}
	err = v.storageClassValidation(descriptorFile)
	if err != nil {
		return err
	}

	if !descriptorFile.ControlPlane.Managed {
		err = v.extraVolumesValidation(descriptorFile.ControlPlane.ExtraVolumes, "controlplane")
		if err != nil {
			return err
		}
		for _, wn := range descriptorFile.WorkerNodes {
			err = v.extraVolumesValidation(wn.ExtraVolumes, "workernodes "+wn.Name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func secretsAzureValidations(secretsFile commons.SecretsFile) error {
	err := commonsSecretsValidations(secretsFile)
	if err != nil {
		return err
	}
	return nil
}

func aksVersionValidation(descriptorFile commons.DescriptorFile, secretsFile commons.SecretsFile) error {
	var availableVersions []string
	var azureSecrets commons.AzureCredentials

	if secretsFile.Secrets.AZURE.Credentials != (commons.AzureCredentials{}) {
		azureSecrets = secretsFile.Secrets.AZURE.Credentials
	} else {
		azureSecrets = descriptorFile.Credentials.AZURE
	}

	creds, err := azidentity.NewClientSecretCredential(azureSecrets.TenantID, azureSecrets.ClientID, azureSecrets.ClientSecret, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(azureSecrets.SubscriptionID, creds, nil)
	if err != nil {
		return err
	}
	res, err := clientFactory.NewManagedClustersClient().ListKubernetesVersions(ctx, descriptorFile.Region, nil)
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
	if !slices.Contains(availableVersions, strings.ReplaceAll(descriptorFile.K8SVersion, "v", "")) {
		a, _ := json.Marshal(availableVersions)
		return errors.New("AKS only supports Kubernetes versions: " + string(a))
	}
	return nil
}

func aksNodesValidation(workerNodes commons.WorkerNodes) error {
	var IsLetter = regexp.MustCompile(`^[a-z0-9]+$`).MatchString
	for _, node := range workerNodes {
		if !IsLetter(node.Name) || len(node.Name) >= 9 {
			return errors.New("node name must be 9 characters or less & contain only lowercase alphanumeric characters")
		}
	}
	return nil
}

func (v *AzureValidator) storageClassValidation(descriptorFile commons.DescriptorFile) error {
	if descriptorFile.StorageClass.EncryptionKey != "" {
		err := v.storageClassKeyFormatValidation(descriptorFile.StorageClass.EncryptionKey)
		if err != nil {
			return errors.New("Error in StorageClass: " + err.Error())
		}
	}
	err := v.storageClassParametersValidation(descriptorFile)
	if err != nil {
		return errors.New("Error in StorageClass: " + err.Error())
	}

	return nil
}

func (v *AzureValidator) storageClassKeyFormatValidation(key string) error {
	regex := regexp.MustCompile(`^/subscriptions/[a-fA-F0-9-]+/resourceGroups/[\w.-]+/providers/Microsoft\.Compute/diskEncryptionSets/[\w.-]+$`)
	if !regex.MatchString(key) {
		return errors.New("Incorrect encryptionKey format. It must have the format /subscriptions/[SUBSCRIPTION_ID]/resourceGroups/[RESOURCE_GROUP]/providers/Microsoft.ManagedIdentity/diskEncryptionSets/[DISK_ENCRYPION_SETS_NAME]")
	}
	return nil
}

func (v *AzureValidator) storageClassParametersValidation(descriptorFile commons.DescriptorFile) error {
	sc := descriptorFile.StorageClass
	err := verifyFields(descriptorFile)
	fstypes := []string{"xfs", "ext3", "ext4", "ext2", "btrfs"}
	if err != nil {
		return err
	}
	if sc.Parameters.SkuName != "" && !slices.Contains(provisionersTypesAzure, sc.Parameters.SkuName) {
		return errors.New("Unsupported skuname: " + sc.Parameters.SkuName)
	}
	if sc.Parameters.FsType != "" && !slices.Contains(fstypes, sc.Parameters.FsType) {
		return errors.New("Unsupported fsType: " + sc.Parameters.FsType + ". Supported types: " + fmt.Sprint(strings.Join(fstypes, ", ")))
	}
	if sc.Parameters.CachingMode != "" && sc.Parameters.SkuName == "PremiumV2_LRS" && sc.Parameters.CachingMode != "none" {
		return errors.New("With skuName: PremiumV2_LRS, CachingMode only can be none")
	}
	if sc.Parameters.DiskEncryptionSetID != "" {
		err := v.storageClassKeyFormatValidation(descriptorFile.StorageClass.Parameters.DiskEncryptionKmsKey)
		if err != nil {
			return err
		}
	}

	if sc.Parameters.Tags != "" {
		tags := strings.Split(sc.Parameters.Tags, ",")
		regex := regexp.MustCompile(`^(\w+|.*)=(\w+|.*)$`)
		for _, tag := range tags {
			if !regex.MatchString(tag) {
				return errors.New("Incorrect labels format. Labels must have the format 'key1=value1,key2=value2'.")
			}
		}
	}

	return nil
}

func (v *AzureValidator) extraVolumesValidation(extraVolumes []commons.ExtraVolume, nodeRole string) error {
	for i, ev := range extraVolumes {
		if ev.Name == "" {
			return errors.New("All  extravolumes must have their own name in " + nodeRole + ".")
		}
		name1 := ev.Name
		for _, ev2 := range extraVolumes[i+1:] {
			if name1 == ev2.Name {
				return errors.New("There can be no more than 1 extravolume with the same name in " + nodeRole + ".")
			}
		}

	}
	return nil
}
