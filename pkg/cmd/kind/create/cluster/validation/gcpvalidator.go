package validation

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
	"sigs.k8s.io/kind/pkg/commons"
)

var gcpInstance *GCPValidator

type GCPValidator struct {
	commonValidator
}

var supportedProvisioners = []string{"pd.csi.storage.gke.io"}

var provisionersTypes = []string{"pd-balanced", "pd-ssd", "pd-standard", "pd-extreme"}

func NewGCPValidator() *GCPValidator {
	if gcpInstance == nil {
		gcpInstance = new(GCPValidator)
	}
	return gcpInstance
}

func (v *GCPValidator) DescriptorFile(descriptorFile commons.DescriptorFile) {
	v.descriptor = descriptorFile
}

func (v *GCPValidator) SecretsFile(secrets commons.SecretsFile) {
	v.secrets = secrets
}

func (v *GCPValidator) Validate(fileType string) error {
	switch fileType {
	case "descriptor":
		err := descriptorGcpValidations((*v).descriptor)
		if err != nil {
			return err
		}
	case "secrets":
		err := secretsGcpValidations((*v).secrets)
		if err != nil {
			return err
		}
	default:
		return errors.New("Incorrect filetype validation")
	}
	return nil
}

func (v *GCPValidator) CommonsValidations() error {
	err := commonsValidations((*v).descriptor, (*v).secrets)
	if err != nil {
		return err
	}
	return nil
}

func descriptorGcpValidations(descriptorFile commons.DescriptorFile) error {
	err := commonsDescriptorValidation(descriptorFile)
	if err != nil {
		return err
	}
	err = storageClassValidation(descriptorFile)
	if err != nil {
		return err
	}
	return nil
}

func secretsGcpValidations(secretsFile commons.SecretsFile) error {
	err := commonsSecretsValidations(secretsFile)
	if err != nil {
		return err
	}
	return nil
}

func storageClassValidation(descriptorFile commons.DescriptorFile) error {

	err := storageClassParametersValidation(descriptorFile)
	if err != nil {
		return errors.New("Error in StorageClass: " + err.Error())
	}

	return nil
}

func storageClassParametersValidation(descriptorFile commons.DescriptorFile) error {
	sc := descriptorFile.StorageClass
	k8s_version := descriptorFile.K8SVersion
	minor, _ := strconv.Atoi(strings.Split(k8s_version, ".")[1])
	err := checkValidParams(descriptorFile)
	if err != nil {
		return err
	}
	if sc.Parameters.Type != "" && !slices.Contains(provisionersTypes, sc.Parameters.Type) {
		return errors.New("Unsupported type: " + sc.Parameters.Type)
	}
	replicationTypeRegex := regexp.MustCompile(`^(none|regional-pd)$`)
	if sc.Parameters.ReplicationType != "" && !replicationTypeRegex.MatchString(sc.Parameters.ReplicationType) {
		return errors.New("Incorrect replication_type. Supported values are none or regional-pd")
	}
	if sc.Parameters.Type == "pd-extreme" && minor < 26 {
		return errors.New("StorageClass Type pd-extreme is only supported by kubernetes versions v1.26.0 and higher")
	}
	if sc.Parameters.Type != "pd-extreme" && sc.Parameters.ProvisionedIopsOnCreate != "" {
		return errors.New("Parameter provisioned_iops_on_create only can be supported for type pd-extreme")
	}

	if sc.Parameters.ProvisionedIopsOnCreate != "" {
		_, err = strconv.Atoi(sc.Parameters.ProvisionedIopsOnCreate)
		if err != nil {
			return errors.New("Parameter provisioned_iops_on_create must be an integer")
		}
	}

	if sc.Parameters.Labels != "" {
		labels := strings.Split(sc.Parameters.Labels, ",")
		regex := regexp.MustCompile(`^(\w+|.*)=(\w+|.*)$`)
		for _, label := range labels {
			if !regex.MatchString(label) {
				return errors.New("Incorrect labels format. Labels must have the format 'key1=value1,key2=value2'")
			}
		}
	}

	return nil
}

func checkValidParams(descriptor commons.DescriptorFile) error {
	err := verifyFields(descriptor)
	if err != nil {
		return err
	}
	return nil
}

func checkEmptyParam(param string, name string) error {
	if param != "" {
		return errors.New("Parameter " + name + "is not supported for gcp provider. The parameters supported are: type, provisioned_iops_on_create, replication_type, disk_encryption_kms_key, labels")
	}
	return nil
}
