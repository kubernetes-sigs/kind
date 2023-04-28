package validation

import (
	"errors"
	"os"

	"sigs.k8s.io/kind/pkg/commons"
)

type Validator interface {
	DescriptorFile(descriptorFile commons.DescriptorFile)
	SecretsFile(secretFile commons.SecretsFile)
	Validate(fileType string) error
	CommonsValidations() error
}

type commonValidator struct {
	descriptor commons.DescriptorFile
	secrets    commons.SecretsFile
}

var validator Validator

func InitValidator(descriptorPath string) error {
	descriptorFile, err := commons.GetClusterDescriptor(descriptorPath)
	if err != nil {
		return err
	}

	infraProvider := descriptorFile.InfraProvider
	managed := descriptorFile.ControlPlane.Managed
	validator, err = getValidator(infraProvider, managed)
	if err != nil {
		return err
	}

	validator.DescriptorFile(*descriptorFile)
	return nil
}

func ExecuteDescriptorValidations() error {

	if err := validator.Validate("descriptor"); err != nil {
		return err
	}
	return nil
}

func ExecuteSecretsValidations(secretsPath string, vaultPassword string) error {
	_, err := os.Stat(secretsPath)
	if err == nil {
		secretsFile, err := commons.GetSecretsFile(secretsPath, vaultPassword)
		if err != nil {
			return err
		}

		validator.SecretsFile(*secretsFile)
		err = validator.Validate("secrets")
		if err != nil {
			return err
		}
	}

	return nil
}

func ExecuteCommonsValidations() error {
	err := validator.CommonsValidations()
	if err != nil {
		return err
	}
	return nil
}

func getValidator(provider string, managed bool) (Validator, error) {
	switch provider {
	case "aws":
		if managed {
			return newEKSValidator(), nil
		}
		return nil, errors.New("WIP in not manage AWS")
	case "azure":
		if managed {
			return newAKSValidator(), nil
		}
		return nil, errors.New("WIP in not manage Azure")
	case "gcp":
		if managed {
			return nil, errors.New("WIP in manage GCP")
		}
		return NewGCPValidator(), nil
	default:
		return nil, errors.New("WIP in others providers")
	}
}
