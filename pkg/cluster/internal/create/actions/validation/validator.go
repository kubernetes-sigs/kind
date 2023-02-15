package validation

import (
	"errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"
)

type Validator interface {
	descriptorFile(descriptorFile cluster.DescriptorFile)
	secretsFile(secretsFile cluster.SecretsFile)
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
