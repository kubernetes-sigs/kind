package validation

import (
	"errors"

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
		descriptorGcpValidations((*v).descriptor)
	case "secrets":
		secretsGcpValidations((*v).secrets)
	default:
		return errors.New("Incorrect filetype validation")
	}
	return nil
}

func descriptorGcpValidations(descriptorFile commons.DescriptorFile) error {
	validateK8sVersion(descriptorFile.K8SVersion)
	return nil
}

func secretsGcpValidations(secretsFile commons.SecretsFile) error {

	return nil
}
