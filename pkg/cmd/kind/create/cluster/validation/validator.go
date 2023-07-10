package validation

import (
	"errors"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kind/pkg/commons"
)

type Validator interface {
	Spec(spec commons.Spec)
	SecretsFile(secretFile commons.SecretsFile)
	Validate(fileType string) error
	CommonsValidations() error
}

type commonValidator struct {
	descriptor commons.Spec
	secrets    commons.SecretsFile
}

var validator Validator

func InitValidator(descriptorPath string) error {
	keosCluster, err := commons.GetClusterDescriptor(descriptorPath)
	if err != nil {
		return err
	}

	infraProvider := keosCluster.Spec.InfraProvider
	managed := keosCluster.Spec.ControlPlane.Managed
	validator, err = getValidator(infraProvider, managed)
	if err != nil {
		return err
	}

	validator.Spec(keosCluster.Spec)
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
		return newAWSValidator(managed), nil
	case "azure":
		return newAzureValidator(managed), nil
	case "gcp":
		if managed {
			return nil, errors.New("WIP in managed GCP")
		}
		return NewGCPValidator(), nil
	default:
		return nil, errors.New("WIP in others providers")
	}
}

func verifyFields(descriptor commons.Spec) error {
	params := descriptor.StorageClass.Parameters
	supportedFields := []string{}
	switch descriptor.InfraProvider {
	case "gcp":
		supportedFields = []string{"type", "provisioned_iops_on_create", "replication_type", "labels"}
		err := verifyAdditionalFields(params, []string{"Type", "ProvisionedIopsOnCreate", "ReplicationType", "Labels"})
		if err != nil {
			return errors.New(err.Error() + "Supported fields for " + descriptor.InfraProvider + ": " + strings.Join(supportedFields, ", "))
		}
	case "aws":
		supportedFields = []string{"type", "iopsPerGB", "fsType", "allowAutoIOPSPerGBIncrease", "iops", "throughput", "blockExpress", "blockSize", "labels"}
		err := verifyAdditionalFields(params, []string{"Type", "IopsPerGB", "FsType", "AllowAutoIOPSPerGBIncrease", "Iops", "Throughput", "BlockExpress", "BlockSize", "Labels"})
		if err != nil {
			return errors.New(err.Error() + "Supported fields for " + descriptor.InfraProvider + ": " + strings.Join(supportedFields, ", "))
		}
	case "azure":
		supportedFields = []string{"provisioner", "fsType", "skuName", "kind", "cachingMode", "diskEncryptionType", "resourceGroup", "tags", "networkAccessPolicy", "publicNetworkAccess", "diskAccessID", "enableBursting", "enablePerformancePlus", "subscriptionID"}
		err := verifyAdditionalFields(params, []string{"Provisioner", "FsType", "SkuName", "Kind", "CachingMode", "DiskEncryptionType", "ResourceGroup", "Tags", "NetworkAccessPolicy", "PublicNetworkAccess", "DiskAccessID", "EnableBursting", "EnablePerformancePlus", "SubscriptionID"})
		if err != nil {
			return errors.New(err.Error() + "Supported fields for " + descriptor.InfraProvider + ": " + strings.Join(supportedFields, ", "))
		}
	}

	return nil
}

func verifyAdditionalFields(obj commons.SCParameters, expectedFields []string) error {
	fields := getFieldNames(obj)

	for _, field := range fields {
		if contains(expectedFields, field) {
			continue
		}

		value := reflect.ValueOf(obj).FieldByName(field)
		expectedValue := reflect.Zero(value.Type()).Interface()

		if !reflect.DeepEqual(value.Interface(), expectedValue) {
			return errors.New("Field not expected " + field + ". ")
		}
	}

	return nil
}

func getFieldNames(obj interface{}) []string {
	var fields []string
	objType := reflect.TypeOf(obj)
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		fields = append(fields, field.Name)
	}
	return fields
}

func contains(list []string, item string) bool {
	for _, element := range list {
		if element == item {
			return true
		}
	}
	return false
}

func structToYAML(data interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(yamlBytes), nil
}
