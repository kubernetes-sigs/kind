package validation

import (
	"errors"

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
		err := descriptorAksValidations((*v).descriptor)
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

func descriptorAksValidations(descriptorFile commons.DescriptorFile) error {
	err := commonsDescriptorValidation(descriptorFile)
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
