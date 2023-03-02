package validation

import (
	"errors"

	"sigs.k8s.io/kind/pkg/commons"
)

var eksInstance *EKSValidator

type EKSValidator struct {
	descriptor commons.DescriptorFile
	secrets    commons.SecretsFile
}

func createEksInstance() *EKSValidator {
	// inicialización del singleton
	return &EKSValidator{}
}

func newEKSValidator() *EKSValidator {
	if eksInstance == nil {
		eksInstance = createEksInstance()
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
		validateEksDescriptor(*v)
	case "secrets":
		validateEksSecrets(*v)
	default:
		return errors.New("Incorrect filetype validation")
	}
	return nil
}

func validateEksDescriptor(v EKSValidator) error {
	descriptorEksValidations(v.descriptor)
	return nil
}

func validateEksSecrets(v EKSValidator) error {
	secretsValidations(v.secrets)
	return nil
}

func descriptorEksValidations(descriptorFile commons.DescriptorFile) error {
	validateK8sVersion(descriptorFile.K8SVersion)
	return nil
}

func secretsValidations(secretsFile commons.SecretsFile) error {

	return nil
}
