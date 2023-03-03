package validation

import (
	"errors"
	"strconv"

	"sigs.k8s.io/kind/pkg/commons"
)

func commonsDescriptorValidation(descriptor commons.DescriptorFile) error {
	err := validateK8sVersion(descriptor.K8SVersion)
	if err != nil {
		return err
	}
	err = validateMaxSizeIsGtMinSize(descriptor.WorkerNodes)
	if err != nil {
		return err
	}
	err = ifBalancedQuantityValidations(descriptor.WorkerNodes)
	if err != nil {
		return err
	}
	err = singleKeosInstaller()
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

func validateK8sVersion(k8sVersion string) error {
	return nil
	//eksctl version -o json | jq -r '.EKSServerSupportedVersions[]'
}

func singleKeosInstaller() error {
	// Cuando se merge refactor credentials. Solo un registry con keos_installer
	// count := 0
	// for _, dr := range dockerRegistries {
	//     if dr.KeosRegistry {
	//         count++
	//         if count > 1 {
	//             return errors.New("There is more than 1 docker_registry defined as keos_registry")
	//         }
	//     }
	// }
	// return nil
	return nil
}

func validateExistsCredentials(descriptor commons.DescriptorFile, secrets commons.SecretsFile) error {
	//Existen credenciales en el secrets o descriptor
	return nil
}

func validateSingleRegistryInDomain() error {
	//Solo un registry por dominio
	return nil
}

func validateRegistryCredentials(descriptor commons.DescriptorFile, secrets commons.SecretsFile) error {
	//Si auth_required=true deben existir las credenciales del registry en secrets o descriptor
	return nil
}

func validateMaxSizeIsGtMinSize(workerNodes commons.WorkerNodes) error {
	for _, wn := range workerNodes {
		minSize := wn.NodeGroupMinSize
		maxSize := wn.NodeGroupMaxSize
		if minSize > maxSize {
			return errors.New("max_size (" + strconv.Itoa(maxSize) + ") must be equal or greater than min_size (" + strconv.Itoa(minSize) + ")")
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
