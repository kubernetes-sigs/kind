package validation

import (
	"errors"
	"reflect"
	"strings"

	"github.com/oleiade/reflections"
	"sigs.k8s.io/kind/pkg/commons"
)

func commonsDescriptorValidation(descriptor commons.DescriptorFile) error {

	err := ifBalancedQuantityValidations(descriptor.WorkerNodes)
	if err != nil {
		return err
	}
	err = singleKeosInstaller(descriptor)
	if err != nil {
		return err
	}
	err = validateUniqueCredentialsRegistry(descriptor.Credentials.DockerRegistries, "descriptor")
	if err != nil {
		return err
	}
	err = validateUniqueRegistry(descriptor.DockerRegistries)
	if err != nil {
		return err
	}
	return nil
}

func commonsValidations(descriptor commons.DescriptorFile, secrets commons.SecretsFile) error {
	err := validateExistsCredentials(descriptor, secrets)
	if err != nil {
		return err
	}
	err = validateRegistryCredentials(descriptor, secrets)
	if err != nil {
		return err
	}
	return nil
}

func commonsSecretsValidations(secrets commons.SecretsFile) error {
	err := validateUniqueCredentialsRegistry(secrets.Secrets.DockerRegistries, "secrets")
	if err != nil {
		return err
	}
	return nil
}

func singleKeosInstaller(descriptor commons.DescriptorFile) error {
	// Solo un registry con keos_installer
	count := 0
	for _, dr := range descriptor.DockerRegistries {
		if dr.KeosRegistry {
			count++
			if count > 1 {
				return errors.New("There is more than 1 docker_registry defined as keos_registry")
			}
		}
	}
	return nil
}

func validateExistsCredentials(descriptor commons.DescriptorFile, secrets commons.SecretsFile) error {
	//Existen credenciales en el secrets o descriptor
	infraProvider := descriptor.InfraProvider
	credentialsProvider, err := reflections.GetField(secrets.Secrets, strings.ToUpper(infraProvider))
	if err != nil || reflect.DeepEqual(credentialsProvider, reflect.Zero(reflect.TypeOf(credentialsProvider)).Interface()) {
		credentialsProvider, err = reflections.GetField(descriptor.Credentials, strings.ToUpper(infraProvider))
		if err != nil || reflect.DeepEqual(credentialsProvider, reflect.Zero(reflect.TypeOf(credentialsProvider)).Interface()) {
			return errors.New("There is not " + infraProvider + " credentials in descriptor or secrets file")
		}
		return nil
	}

	return nil
}

func validateSingleRegistryInDomain() error {
	//Solo un registry por dominio
	return nil
}

func validateRegistryCredentials(descriptor commons.DescriptorFile, secrets commons.SecretsFile) error {
	//Si auth_required=true deben existir las credenciales del registry en secrets o descriptor
	for _, dockerRegistry := range descriptor.DockerRegistries {
		if dockerRegistry.AuthRequired {

			for _, dockerRegistryCredential := range secrets.Secrets.DockerRegistries {
				if dockerRegistryCredential.URL == dockerRegistry.URL {
					continue
				}
			}
			for _, dockerRegistryCredential := range descriptor.Credentials.DockerRegistries {
				if dockerRegistryCredential.URL == dockerRegistry.URL {
					continue
				}
			}
			return errors.New("There is no credential in either the descriptor or the secret for the url: " + dockerRegistry.URL)
		}
	}
	return nil
}

func ifBalancedQuantityValidations(workerNodes commons.WorkerNodes) error {
	for _, wn := range workerNodes {
		if wn.ZoneDistribution == "balanced" {
			if wn.Quantity < 3 {
				return errors.New("Quantity in WorkerNodes " + wn.Name + ", must be equal or greater than 3 when HA is required")
			}
		}
	}
	return nil
}

func validateUniqueCredentialsRegistry(dockerRegistries []commons.DockerRegistryCredentials, fileName string) error {
	for _, c1 := range dockerRegistries {
		for _, c2 := range dockerRegistries {
			if c1.URL == c2.URL {
				return errors.New("There is more than one credential for the registry: " + c1.URL + ", in file: " + fileName)
			}
		}
	}
	return nil
}

func validateUniqueRegistry(dockerRegistries []commons.DockerRegistry) error {
	for _, c1 := range dockerRegistries {
		for _, c2 := range dockerRegistries {
			if c1.URL == c2.URL {
				return errors.New("There is more than one docker_registry with url: " + c1.URL)
			}
		}
	}
	return nil
}

//az de subnets vs az workers
