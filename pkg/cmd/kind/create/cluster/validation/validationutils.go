package validation

import (
	"reflect"
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
