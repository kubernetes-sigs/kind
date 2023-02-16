package validation

import (
	"errors"
	"strings"
)

type EKSValidator struct {
	descriptor map[string]interface{}
	secrets    map[string]interface{}
}

func newEKSValidator() *EKSValidator {
	return &EKSValidator{}
}

func (v *EKSValidator) descriptorFile(descriptor map[string]interface{}) {
	v.descriptor = descriptor
}

func (v *EKSValidator) secretsFile(secrets map[string]interface{}) {
	v.secrets = secrets
}

func (v *EKSValidator) validate(fileType string) error {
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
	//panic("UnImplemented")
	descriptorValidations(v.descriptor)
	return nil
}

func validateSecrets(v EKSValidator) error {
	//panic("UnImplemented")
	secretsValidations(v.secrets)
	return nil
}

func descriptorValidations(descriptorFile map[string]interface{}) error {
	k8s_version_validator(descriptorFile["k8s_version"].(string))
	return nil
}

func k8s_version_validator(k8sVersion string) error {
	start_with_v := strings.HasPrefix("v", k8sVersion)

	if !start_with_v {
		return nil
	}
	return nil
}

func secretsValidations(secretsFile map[string]interface{}) error {

	return nil
}
