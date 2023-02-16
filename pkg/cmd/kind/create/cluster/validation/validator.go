package validation

import (
	"errors"
)

// type Validator interface {
// 	descriptorFile(descriptorFile cluster.DescriptorFile)
// 	secretsFile(secretsFile cluster.SecretsFile)
// 	validate() error
// }

type Validator interface {
	descriptorFile(descriptor map[string]interface{})
	secretsFile(descriptor map[string]interface{})
	validate(fileType string) error
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
