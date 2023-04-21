package validation

import (
	"errors"

	"sigs.k8s.io/kind/pkg/commons"
)

var eksInstance *EKSValidator

type EKSValidator struct {
	commonValidator
}

func newEKSValidator() *EKSValidator {
	if eksInstance == nil {
		eksInstance = new(EKSValidator)
	}
	return eksInstance
}

func (v *EKSValidator) DescriptorFile(descriptorFile commons.DescriptorFile) {
	v.descriptor = descriptorFile
}

func (v *EKSValidator) SecretsFile(secrets commons.SecretsFile) {
	v.secrets = secrets
}

func (v *EKSValidator) Validate(fileType string) error {
	switch fileType {
	case "descriptor":
		err := descriptorEksValidations((*v).descriptor)
		if err != nil {
			return err
		}
	case "secrets":
		err := secretsEksValidations((*v).secrets)
		if err != nil {
			return err
		}
	default:
		return errors.New("Incorrect filetype validation")
	}
	return nil
}

func (v *EKSValidator) CommonsValidations() error {
	err := commonsValidations((*v).descriptor, (*v).secrets)
	if err != nil {
		return err
	}
	return nil
}

func descriptorEksValidations(descriptorFile commons.DescriptorFile) error {
	err := commonsDescriptorValidation(descriptorFile)
	if err != nil {
		return err
	}
	return nil
}

func secretsEksValidations(secretsFile commons.SecretsFile) error {
	err := commonsSecretsValidations(secretsFile)
	if err != nil {
		return err
	}
	return nil
}
