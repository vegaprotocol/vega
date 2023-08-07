package inspecttx

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/iancoleman/strcase"

	proto2 "github.com/golang/protobuf/proto"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/golang/protobuf/jsonpb"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/proto"

	"github.com/nsf/jsondiff"
)

var JsonMarshaller = jsonpb.Marshaler{
	Indent:      "   ",
	EnumsAsInts: false,
}

// ComparableJson used to contain json data for comparison.
type ComparableJson struct {
	OriginalJson json.RawMessage
	CoreJson     json.RawMessage
	DiffType     DiffType
}

func marshalTransactionAndInputDataToString(transaction *commandspb.Transaction, inputData *commandspb.InputData) (string, string, error) {
	marshalledTransaction, err := MarshalToJSONWithOneOf(transaction)
	if err != nil {
		return "", "", fmt.Errorf("couldn't marshal transaction: %w", err)
	}

	marshalledInputData, err := MarshalToJSONWithOneOf(inputData)
	if err != nil {
		return "", "", fmt.Errorf("couldn't marshal input data: %w", err)
	}

	return marshalledTransaction, marshalledInputData, nil
}

// MarshalToJSONWithOneOf this method exists to accommodate to marshalling proto fields with a 'oneof' tag. These are ignored in marshalling unless explicitly handled.
func MarshalToJSONWithOneOf(pb proto2.Message) (string, error) {
	marshalledTransaction, err := JsonMarshaller.MarshalToString(pb)
	if err != nil {
		return "", fmt.Errorf("error marshalling the proto message to a string\nerr: %d", err)
	}

	transactionValsWithoutOneOfFields := map[string]interface{}{}
	oneOfValues := map[string]interface{}{}
	err = json.Unmarshal([]byte(marshalledTransaction), &transactionValsWithoutOneOfFields)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling transaction struct to a map.\nerr: %v", err)
	}

	pbType := reflect.TypeOf(pb).Elem()
	pbValue := reflect.ValueOf(pb).Elem()

	for i := 0; i < pbType.NumField(); i++ {
		field := pbType.Field(i)

		if field.Tag.Get("protobuf_oneof") != "" {
			oneOfFieldName := field.Tag.Get("protobuf_oneof")
			oneOfField := pbValue.FieldByName(field.Name)

			if oneOfField.IsValid() && !oneOfField.IsNil() {
				oneOfValue := getInterfaceValue(oneOfField)
				if oneOfValue != nil {
					oneOfSelectedType := reflect.TypeOf(oneOfValue)
					oneOfSelectedValue := reflect.ValueOf(oneOfValue).Elem().Field(0).Interface()

					jsonKey := strcase.ToLowerCamel(oneOfSelectedType.Elem().Field(0).Name)
					oneOfData := map[string]interface{}{jsonKey: oneOfSelectedValue}
					oneOfValues[oneOfFieldName] = oneOfData
					delete(transactionValsWithoutOneOfFields, jsonKey)
				}
			}
		}
	}

	combinedWithOneOfVals := make(map[string]interface{})
	for key, value := range transactionValsWithoutOneOfFields {
		combinedWithOneOfVals[key] = value
	}

	for key, value := range oneOfValues {
		combinedWithOneOfVals[key] = value
	}

	mergedJson, err := json.Marshal(combinedWithOneOfVals)
	if err != nil {
		return "", fmt.Errorf("Error merging original json vals with 'oneof' fields to JSON:\nerr: %v", err)
	}

	return string(mergedJson), nil
}

func getInterfaceValue(v reflect.Value) interface{} {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		return getInterfaceValue(v.Elem())
	}

	if v.Kind() == reflect.Interface {
		return v.Interface()
	}

	return nil
}

func unmarshalTransaction(decodedTransactionBytes []byte) (*commandspb.Transaction, *commandspb.InputData, error) {
	unmarshalledTransaction := &commandspb.Transaction{}
	unmarshalledInputData := &commandspb.InputData{}
	if err := proto.Unmarshal(decodedTransactionBytes, unmarshalledTransaction); err != nil {
		return unmarshalledTransaction, unmarshalledInputData, fmt.Errorf("unable to unmarshal transaction: %w", err)
	}

	inputData, err := commands.UnmarshalInputData(unmarshalledTransaction.InputData)
	if err != nil {
		return unmarshalledTransaction, unmarshalledInputData, fmt.Errorf("unable to unmarshal input data: %w", err)
	}

	return unmarshalledTransaction, inputData, nil
}

func compareJson(firstJson []byte, secondJson []byte) (jsondiff.Difference, string) {
	options := jsondiff.DefaultHTMLOptions()
	result, diffHtml := jsondiff.Compare(firstJson, secondJson, &options)
	return result, diffHtml
}
