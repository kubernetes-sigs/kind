package validation

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
