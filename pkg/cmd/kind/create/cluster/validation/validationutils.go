package validation

import (
	"errors"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kind/pkg/commons"
)

func StructToMap(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Obtener el tipo y valor del struct
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	// Asegurarse de que obj sea un struct
	if t.Kind() == reflect.Struct {
		// Recorrer los campos del struct
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldValue := v.Field(i)

			// Obtener el nombre del campo y el valor
			fieldName := field.Name
			fieldValueInterface := fieldValue.Interface()

			// Agregar el campo y valor al mapa resultante
			result[fieldName] = fieldValueInterface
		}
	}

	return result
}

func verifyFields(descriptor commons.DescriptorFile) error {
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
		supportedFields = []string{"type", "iopsPerGB", "fstype", "allowAutoIOPSPerGBIncrease", "iops", "throughput", "blockExpress", "blockSize"}
		err := verifyAdditionalFields(params, []string{"Type", "IopsPerGB", "FsType", "AllowAutoIOPSPerGBIncrease", "Iops", "Throughput", "BlockExpress", "BlockSize"})
		if err != nil {
			return errors.New(err.Error() + "Supported fields for " + descriptor.InfraProvider + ": " + strings.Join(supportedFields, ", "))
		}
	case "azure":
		supportedFields = []string{"provisioner", "skuName", "kind", "cachingMode", "diskEncryptionType", "resourceGroup", "tags", "mountOptions"}
		err := verifyAdditionalFields(params, []string{"Provisioner", "SkuName", "Kind", "CachingMode", "DiskEncryptionType", "ResourceGroup", "Tags", "MountOptions"})
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
