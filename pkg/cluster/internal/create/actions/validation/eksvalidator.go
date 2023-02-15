package validation

import "sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"

type EKSValidator struct {
	descriptor cluster.DescriptorFile
	secrets    cluster.SecretsFile
}

func newEKSValidator() *EKSValidator {
	return &EKSValidator{}
}

func (v *EKSValidator) descriptorFile(descriptor cluster.DescriptorFile) {
	v.descriptor = descriptor
}

func (v *EKSValidator) secretsFile(secrets cluster.SecretsFile) {
	v.secrets = secrets
}

func (v *EKSValidator) validate() error {
	//panic("UnImplemented")

	return nil
}
