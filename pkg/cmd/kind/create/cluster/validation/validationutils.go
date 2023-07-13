package validation

import (
	"reflect"
)

func StructToMap(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldValue := v.Field(i)

			fieldName := field.Name
			fieldValueInterface := fieldValue.Interface()

			result[fieldName] = fieldValueInterface
		}
	}

	return result
}
