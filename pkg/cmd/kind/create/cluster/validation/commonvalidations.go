package validation

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/oleiade/reflections"
	"sigs.k8s.io/kind/pkg/commons"
)

func commonsDescriptorValidation(spec commons.Spec) error {

	var err error

	err = quantityValidations(spec.WorkerNodes)
	if err != nil {
		return err
	}
	err = singleKeosInstaller(spec)
	if err != nil {
		return err
	}
	err = validateUniqueCredentialsRegistry(spec.Credentials.DockerRegistries, "descriptor")
	if err != nil {
		return err
	}
	err = validateUniqueRegistry(spec.DockerRegistries)
	if err != nil {
		return err
	}
	err = validateTaintsFormat(spec.WorkerNodes)
	if err != nil {
		return err
	}
	return nil
}

func commonsValidations(spec commons.Spec, secrets commons.SecretsFile) error {
	err := validateExistsCredentials(spec, secrets)
	if err != nil {
		return err
	}
	err = validateRegistryCredentials(spec, secrets)
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

func singleKeosInstaller(descriptor commons.Spec) error {
	// Only one registry with keos_installer
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

func validateExistsCredentials(spec commons.Spec, secrets commons.SecretsFile) error {
	// Credentials must exist in the secrets or descriptor
	infraProvider := spec.InfraProvider
	credentialsProvider, err := reflections.GetField(secrets.Secrets, strings.ToUpper(infraProvider))
	if err != nil || reflect.DeepEqual(credentialsProvider, reflect.Zero(reflect.TypeOf(credentialsProvider)).Interface()) {
		credentialsProvider, err = reflections.GetField(spec.Credentials, strings.ToUpper(infraProvider))
		if err != nil || reflect.DeepEqual(credentialsProvider, reflect.Zero(reflect.TypeOf(credentialsProvider)).Interface()) {
			return errors.New("There is not " + infraProvider + " credentials in descriptor or secrets file")
		}
		return nil
	}

	return nil
}

func validateRegistryCredentials(spec commons.Spec, secrets commons.SecretsFile) error {
	//If auth_required=true, the registry credentials must exist in secrets or descriptor.
	for _, dockerRegistry := range spec.DockerRegistries {
		if dockerRegistry.AuthRequired {

			existCredentials := false
			for _, dockerRegistryCredential := range secrets.Secrets.DockerRegistries {
				if dockerRegistryCredential.URL == dockerRegistry.URL {
					existCredentials = true
					break
				}
			}
			if !existCredentials {
				for _, dockerRegistryCredential := range spec.Credentials.DockerRegistries {
					if dockerRegistryCredential.URL == dockerRegistry.URL {
						existCredentials = true
						break
					}
				}
			}
			if existCredentials {
				continue
			}
			return errors.New("There is no credential in either the descriptor or the secret for the registry with url: " + dockerRegistry.URL)
		}
	}
	return nil
}

func quantityValidations(workerNodes commons.WorkerNodes) error {
	for _, wn := range workerNodes {
		// Cluster Autoscaler doesn't scale a managed node group lower than minSize or higher than maxSize.
		if wn.NodeGroupMaxSize < wn.Quantity && wn.NodeGroupMaxSize != 0 {
			return errors.New("max_size in WorkerNodes " + wn.Name + ", must be equal or greater than quantity")
		}
		if wn.Quantity < wn.NodeGroupMinSize {
			return errors.New("quantity in WorkerNodes " + wn.Name + ", must be equal or greater than min_size")
		}
		if wn.NodeGroupMinSize < 0 {
			return errors.New("min_size in WorkerNodes " + wn.Name + ", must be equal or greater than 0")
		}
		if wn.ZoneDistribution == "balanced" || wn.ZoneDistribution == "" {
			if wn.AZ != "" {
				return errors.New("az in WorkerNodes " + wn.Name + ", can not be set when HA is required")
			}
			if wn.Quantity < 3 {
				return errors.New("quantity in WorkerNodes " + wn.Name + ", must be equal or greater than 3 when HA is required")
			}
		}
	}
	return nil
}

func validateUniqueCredentialsRegistry(dockerRegistries []commons.DockerRegistryCredentials, fileName string) error {
	for i, c1 := range dockerRegistries {
		for j, c2 := range dockerRegistries {

			if i == j {
				continue
			}
			if c1.URL == c2.URL {
				return errors.New("There is more than one credential for the registry: " + c1.URL + ", in file: " + fileName)
			}
		}
	}
	return nil
}

func validateUniqueRegistry(dockerRegistries []commons.DockerRegistry) error {
	for i, c1 := range dockerRegistries {
		for j, c2 := range dockerRegistries {
			if i == j {
				continue
			}
			if c1.URL == c2.URL {
				return errors.New("There is more than one docker_registry with url: " + c1.URL)
			}
		}
	}
	return nil
}

func validateTaintsFormat(wns commons.WorkerNodes) error {
	regex := regexp.MustCompile(`^(\w+|.*)=(\w+|.*):(NoSchedule|PreferNoSchedule|NoExecute)$`)
	for _, wn := range wns {
		for i, taint := range wn.Taints {
			if !regex.MatchString(taint) {
				return errors.New("Incorrect taint format in taint[" + strconv.Itoa(i) + "] of wn: " + wn.Name + "")
			}
		}
	}

	return nil
}

func validateWnAZWithSubnetsAZ() {
	// az de subnets vs az workers
	// Cuando se mergee VPC custom
}
