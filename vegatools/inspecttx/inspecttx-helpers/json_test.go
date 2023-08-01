package inspecttx_helpers

import (
	"testing"

	"code.vegaprotocol.io/vega/libs/proto"

	"github.com/nsf/jsondiff"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func TestMarshalTransactionAndInputDataToString(t *testing.T) {
	transaction := &commandspb.Transaction{
		// Your test data here
	}
	inputData := &commandspb.InputData{
		// Your test data here
	}

	marshalledTransaction, marshalledInputData, err := marshalTransactionAndInputDataToString(transaction, inputData)
	if err != nil {
		t.Fatalf("Error marshalling transaction and input data: %v", err)
	}

	// Test if the returned marshalled strings are not empty or invalid JSON
	if len(marshalledTransaction) == 0 {
		t.Errorf("Marshalled transaction is empty")
	}
	if len(marshalledInputData) == 0 {
		t.Errorf("Marshalled input data is empty")
	}

	// You can also further test the contents of the marshalled strings if needed
}

func TestUnmarshalTransaction(t *testing.T) {
	transaction := &commandspb.Transaction{
		// Your test data here
	}

	// Marshal the transaction to bytes for testing
	encodedTransaction, err := proto.Marshal(transaction)
	if err != nil {
		t.Fatalf("Error marshalling transaction: %v", err)
	}

	unmarshalledTransaction, inputData, err := unmarshalTransaction(encodedTransaction)
	if err != nil {
		t.Fatalf("Error unmarshalling transaction: %v", err)
	}

	// Test if the unmarshalled transaction and input data are not nil
	if unmarshalledTransaction == nil {
		t.Errorf("Unmarshalled transaction is nil")
	}
	if inputData == nil {
		t.Errorf("Unmarshalled input data is nil")
	}

	// You can also further test the contents of the unmarshalled structs if needed
}

func TestCompareJson(t *testing.T) {
	firstJson := []byte(`{"name": "John", "age": 30}`)
	secondJson := []byte(`{"name": "John", "age": 300}`)

	result, diffHtml := compareJson(firstJson, secondJson)

	if result != jsondiff.NoMatch {
		t.Errorf("Expected JSONs to be different, but got: %v", result)
	}

	if len(diffHtml) == 0 {
		t.Errorf("Diff HTML is empty")
	}
}
