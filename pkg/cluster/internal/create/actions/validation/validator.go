package validation

import (
	"errors"
)

type Validator interface {
	setDescriptorPath(descriptorpath string)
	setSecretsPath(secretPath string)
	validate() error
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
