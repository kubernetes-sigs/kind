package validation

import (
	"errors"
	"fmt"

	"sigs.k8s.io/kind/pkg/commons"
)

var gcpInstance *GCPValidator

type GCPValidator struct {
	descriptor commons.DescriptorFile
	secrets    commons.SecretsFile
}

func createGCPInstance() *GCPValidator {
	return &GCPValidator{}
}

func NewGCPValidator() *GCPValidator {
	fmt.Println("GCPValidator")
	if gcpInstance == nil {
		gcpInstance = createGCPInstance()
	}
	return gcpInstance
}

func (v *GCPValidator) DescriptorFile(descriptorFile commons.DescriptorFile) {
	v.descriptor = descriptorFile
}

func (v *GCPValidator) SecretsFile(secrets commons.SecretsFile) {
	v.secrets = secrets
}

func (v *GCPValidator) Validate(fileType string) error {
	switch fileType {
	case "descriptor":
		err := descriptorGcpValidations((*v).descriptor)
		if err != nil {
			return err
		}
	case "secrets":
		err := secretsGcpValidations((*v).secrets)
		if err != nil {
			return err
		}
	default:
		return errors.New("Incorrect filetype validation")
	}
	return nil
}

func (v *GCPValidator) CommonsValidations() error {
	err := commonsValidations((*v).descriptor, (*v).secrets)
	if err != nil {
		return err
	}
	return nil
}

func descriptorGcpValidations(descriptorFile commons.DescriptorFile) error {
	err := commonsDescriptorValidation(descriptorFile)
	if err != nil {
		return err
	}
	return nil
}

func secretsGcpValidations(secretsFile commons.SecretsFile) error {

	return nil
}
