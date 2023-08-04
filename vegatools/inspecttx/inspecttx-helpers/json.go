package inspecttx_helpers

import (
	"encoding/json"
	"fmt"

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

type ComparableJson struct {
	OriginalJson json.RawMessage
	CoreJson     json.RawMessage
	DiffType     DiffType
}

func marshalTransactionAndInputDataToString(transaction *TransactionAlias, inputData *commandspb.InputData) (string, string, error) {
	marshalledTransaction, err := transaction.MarshalJSON()
	if err != nil {
		return "", "", fmt.Errorf("couldn't marshal transaction: %w", err)
	}

	marshalledInputData, err := JsonMarshaller.MarshalToString(inputData)
	if err != nil {
		return "", "", fmt.Errorf("couldn't marshal input data: %w", err)
	}

	return string(marshalledTransaction), marshalledInputData, nil
}

type TransactionAlias struct {
	*commandspb.Transaction
	*commandspb.InputData
}

// MarshalJSON used to accommodate for 'oneof' types that would otherwise be left out of marshalling to json, therefore causing unnecessary diffs
// this method checks the oneof types and ensures they are added to json
func (mt *TransactionAlias) MarshalJSON() ([]byte, error) {
	// Convert the embedded struct to JSON.
	data, err := JsonMarshaller.MarshalToString(mt.Transaction)
	if err != nil {
		return nil, err
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return nil, err
	}

	switch v := mt.From.(type) {
	case *commandspb.Transaction_Address:
		jsonData["from"] = map[string]string{"address": v.Address}
		delete(jsonData, "address")
	case *commandspb.Transaction_PubKey:
		jsonData["from"] = map[string]string{"pubKey": v.PubKey}
		delete(jsonData, "pubKey")
	}

	return json.Marshal(jsonData)
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
