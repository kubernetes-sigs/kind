package validation

import (
	"errors"

	"sigs.k8s.io/kind/pkg/commons"
)

type Validator interface {
	DescriptorFile(descriptorFile commons.DescriptorFile)
	SecretsFile(secretFile commons.SecretsFile)
	Validate(fileType string) error
}

func getValidator(provider string, managed bool) (Validator, error) {

	switch provider {
	case "aws":
		if managed {
			return newEKSValidator(), nil
		}
		return nil, errors.New("WIP in not manage AWS")
	default:
		return nil, errors.New("WIP in GCP and others providers")
	}

}
