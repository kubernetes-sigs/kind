package validation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v3"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/kind/pkg/commons"
)

var aksInstance *AKSValidator

type AKSValidator struct {
	commonValidator
}

func newAKSValidator() *AKSValidator {
	if aksInstance == nil {
		aksInstance = new(AKSValidator)
	}
	return aksInstance
}

func (v *AKSValidator) DescriptorFile(descriptorFile commons.DescriptorFile) {
	v.descriptor = descriptorFile
}

func (v *AKSValidator) SecretsFile(secrets commons.SecretsFile) {
	v.secrets = secrets
}

func (v *AKSValidator) Validate(fileType string) error {
	switch fileType {
	case "descriptor":
		err := descriptorAksValidations((*v).descriptor, (*v).secrets)
		if err != nil {
			return err
		}
	case "secrets":
		err := secretsAksValidations((*v).secrets)
		if err != nil {
			return err
		}
	default:
		return errors.New("Incorrect filetype validation")
	}
	return nil
}

func (v *AKSValidator) CommonsValidations() error {
	err := commonsValidations((*v).descriptor, (*v).secrets)
	if err != nil {
		return err
	}
	return nil
}

func descriptorAksValidations(descriptorFile commons.DescriptorFile, secretsFile commons.SecretsFile) error {
	err := commonsDescriptorValidation(descriptorFile)
	if err != nil {
		return err
	}
	err = k8sAksVersionValidation(descriptorFile.K8SVersion, descriptorFile.Region, secretsFile)
	if err != nil {
		return err
	}
	return nil
}

func secretsAksValidations(secretsFile commons.SecretsFile) error {
	err := commonsSecretsValidations(secretsFile)
	if err != nil {
		return err
	}
	return nil
}

func k8sAksVersionValidation(k8sVersion string, region string, secretsFile commons.SecretsFile) error {
	var availableVersions []string

	azureSecrets := secretsFile.Secrets.AZURE.Credentials

	creds, err := azidentity.NewClientSecretCredential(azureSecrets.TenantID, azureSecrets.ClientID, azureSecrets.ClientSecret, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(azureSecrets.SubscriptionID, creds, nil)
	if err != nil {
		return err
	}
	res, err := clientFactory.NewManagedClustersClient().ListKubernetesVersions(ctx, region, nil)
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
	if !slices.Contains(availableVersions, strings.ReplaceAll(k8sVersion, "v", "")) {
		a, _ := json.Marshal(availableVersions)
		return errors.New("AKS only supports Kubernetes versions: " + string(a))
	}
	return nil
}
