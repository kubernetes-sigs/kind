package validation

import (
	"errors"
	"strings"

	"sigs.k8s.io/kind/pkg/commons"
)

var instance *EKSValidator

type EKSValidator struct {
	descriptor commons.DescriptorFile
	secrets    commons.SecretsFile
}

func createInstance() *EKSValidator {
	// inicialización del singleton
	return &EKSValidator{}
}

func newEKSValidator() *EKSValidator {
	if instance == nil {
		instance = createInstance()
	}
	return instance
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
		validateDescriptor(*v)
	case "secrets":
		validateSecrets(*v)
	default:
		return errors.New("Incorrect filetype validation")
	}
	return nil
}

func validateDescriptor(v EKSValidator) error {
	descriptorValidations(v.descriptor)
	return nil
}

func validateSecrets(v EKSValidator) error {
	secretsValidations(v.secrets)
	return nil
}

func descriptorValidations(descriptorFile commons.DescriptorFile) error {
	k8s_version_validator(descriptorFile.K8SVersion)
	return nil
}

func k8s_version_validator(k8sVersion string) error {
	start_with_v := strings.HasPrefix("v", k8sVersion)

	if !start_with_v {
		return nil
	}
	return nil
}

func secretsValidations(secretsFile commons.SecretsFile) error {

	return nil
}
