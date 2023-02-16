package validation

func validateK8sVersion(k8sVersion string) error {
	return nil
	//eksctl version -o json | jq -r '.EKSServerSupportedVersions[]'
}
