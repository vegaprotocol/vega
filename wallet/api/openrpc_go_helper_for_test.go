package api_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func parseASTFromGo(t *testing.T, i interface{}) (astNode, error) {
	t.Helper()

	if i == nil {
		return astNode{}, nil
	}

	rootType := reflect.TypeOf(i)
	nestedProperties, err := resolveNestedProperties(rootType)
	if err != nil {
		return astNode{}, fmt.Errorf("could not reflect the AST for struct %q: %w", rootType.String(), err)
	}

	return astNode{
		nodeType:         nodeTypeObject,
		nestedProperties: nestedProperties,
	}, nil
}

func resolveNestedProperties(rootType reflect.Type) ([]astNode, error) {
	unwrappedType := unwrapType(rootType)

	if unwrappedType.Kind() == reflect.Interface {
		return nil, nil
	}

	if unwrappedType.Kind() == reflect.Array || unwrappedType.Kind() == reflect.Slice {
		return resolveNestedPropertiesForArray(unwrappedType)
	}

	if unwrappedType.Kind() == reflect.Struct && !structShouldBeString(unwrappedType) {
		return resolveNestedPropertiesForStruct(unwrappedType)
	}

	if unwrappedType.Kind() == reflect.Map {
		return resolveNestedPropertiesForMap(unwrappedType)
	}

	return nil, nil
}

func structShouldBeString(unwrappedType reflect.Type) bool {
	return unwrappedType.String() == "time.Time"
}

func resolveNestedPropertiesForMap(unwrappedType reflect.Type) ([]astNode, error) {
	elemType := unwrappedType.Elem()
	unwrappedElemType := unwrapType(elemType)

	jsonTypeForElem, err := goTypeToJSONType(unwrappedElemType)
	if err != nil {
		return nil, fmt.Errorf("could not figure out the type for element field %q: %w", elemType.String(), err)
	}

	nestedPropertiesForField, err := resolveNestedProperties(unwrappedElemType)
	if err != nil {
		return nil, fmt.Errorf("could not resolve nested properties for field %q: %w", unwrappedElemType.String(), err)
	}

	return []astNode{{
		nodeType:         jsonTypeForElem,
		nestedProperties: nestedPropertiesForField,
	}}, nil
}

func resolveNestedPropertiesForArray(unwrappedType reflect.Type) ([]astNode, error) {
	elemType := unwrappedType.Elem()
	unwrappedElemType := unwrapType(elemType)

	if unwrappedElemType.Kind() == reflect.Struct && !structShouldBeString(unwrappedType) {
		itemsProperty, err := resolveNestedPropertiesForStruct(unwrappedElemType)
		if err != nil {
			return nil, fmt.Errorf("could not reflect on the item properties %q: %w", unwrappedType.String(), err)
		}
		return []astNode{{
			nodeType:         nodeTypeObject,
			nestedProperties: itemsProperty,
		}}, nil
	}

	jsonTypeForElem, err := goTypeToJSONType(unwrappedElemType)
	if err != nil {
		return nil, fmt.Errorf("could not figure out the type for element field %q: %w", elemType.String(), err)
	}

	// No name, nor nested properties.
	return []astNode{{nodeType: jsonTypeForElem}}, nil
}

func resolveNestedPropertiesForStruct(rootType reflect.Type) ([]astNode, error) {
	numField := rootType.NumField()
	nestedProperties := make([]astNode, 0, numField)
	numNodesToAccountFor := numField

	for fieldIdx := 0; fieldIdx < numField; fieldIdx++ {
		field := rootType.Field(fieldIdx)

		jsonName, shouldBeAccountedFor, err := retrieveFieldName(field)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve the JSON name for field %q at index %d: %w", field.Name, fieldIdx, err)
		}
		if !shouldBeAccountedFor {
			numNodesToAccountFor--
			continue
		}

		unwrappedType := unwrapType(field.Type)

		jsonType, err := goTypeToJSONType(unwrappedType)
		if err != nil {
			return nil, fmt.Errorf("could not figure out the type for field %q at index %d: %w", field.Name, fieldIdx, err)
		}

		var nestedPropertiesForField []astNode

		if jsonType == nodeTypeArray || jsonType == nodeTypeObject {
			nestedPropertiesForField, err = resolveNestedProperties(unwrappedType)
			if err != nil {
				return nil, fmt.Errorf("could not resolve nested properties for field %q at index %d: %w", field.Name, fieldIdx, err)
			}
		}

		nestedProperties = append(nestedProperties, astNode{
			name:             jsonName,
			nodeType:         jsonType,
			nestedProperties: nestedPropertiesForField,
		})
	}

	return deterministNestedProperties(nestedProperties[:numNodesToAccountFor]), nil
}

func unwrapType(field reflect.Type) reflect.Type {
	if field.Kind() == reflect.Pointer {
		return field.Elem()
	}
	return field
}

func retrieveFieldName(field reflect.StructField) (string, bool, error) {
	protobufValue, exist := field.Tag.Lookup("protobuf")
	if exist {
		names := strings.Split(protobufValue, ",")
		for _, name := range names {
			if strings.HasPrefix(name, "json=") {
				return name[5:], true, nil
			}
		}
	}

	protobufOneofValue, exist := field.Tag.Lookup("protobuf_oneof")
	if exist {
		return protobufOneofValue, true, nil
	}

	jsonValue, exist := field.Tag.Lookup("json")
	if !exist {
		if field.IsExported() {
			return "", false, fmt.Errorf("field is exported but does not have a JSON tag")
		}

		// No json tag, so it is not meant to be used in the API.
		return "", false, nil
	}

	// the first value is the name in the JSON tag
	jsonName := strings.Split(jsonValue, ",")[0]

	if strings.ToLower(field.Name) != strings.ToLower(jsonName) { //nolint:staticcheck
		return "", false, fmt.Errorf("field name %q does not match JSON name %q", field.Name, jsonName)
	}

	return jsonName, true, nil
}

func goTypeToJSONType(fieldType reflect.Type) (nodeType, error) {
	switch fieldType.Kind() {
	case reflect.String:
		return nodeTypeString, nil
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		return nodeTypeNumber, nil
	case reflect.Bool:
		return nodeTypeBoolean, nil
	case reflect.Struct, reflect.Map, reflect.Interface:
		if structShouldBeString(fieldType) {
			return nodeTypeString, nil
		}
		return nodeTypeObject, nil
	case reflect.Array, reflect.Slice:
		if fieldType.Elem().Kind() == reflect.Uint8 {
			return nodeTypeString, nil
		}
		return nodeTypeArray, nil
	default:
		return nodeTypeUnknown, fmt.Errorf("struct type %q cannot be converted to node type", fieldType.Kind().String())
	}
}
