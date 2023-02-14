package validation

type EKSValidator struct {
	descriptorPath string
	secretsPath    string
}

func newEKSValidator() *EKSValidator {
	return &EKSValidator{}
}

func (v *EKSValidator) setDescriptorPath(descriptorPath string) {
	v.descriptorPath = descriptorPath
}

func (v *EKSValidator) setSecretsPath(secretsPath string) {
	v.secretsPath = secretsPath
}

func (v *EKSValidator) validate() error {
	panic("UnImplemented")
}
