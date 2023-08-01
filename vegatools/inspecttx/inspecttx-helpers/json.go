package inspecttx_helpers

import (
	"encoding/json"
	"fmt"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/nsf/jsondiff"
)

type ComparableJson struct {
	OriginalJson json.RawMessage
	CoreJson     json.RawMessage
	DiffType     DiffType
}

func marshalTransactionAndInputDataToString(transaction *commandspb.Transaction, inputData *commandspb.InputData) (string, string, error) {
	marshalledTransaction, err := jsonMarshaller.MarshalToString(transaction)
	if err != nil {
		return "", "", fmt.Errorf("couldn't marshal transaction: %w", err)
	}

	marshalledInputData, err := jsonMarshaller.MarshalToString(inputData)
	if err != nil {
		return "", "", fmt.Errorf("couldn't marshal input data: %w", err)
	}

	return marshalledTransaction, marshalledInputData, nil
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
