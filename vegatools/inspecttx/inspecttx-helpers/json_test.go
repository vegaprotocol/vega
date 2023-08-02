package inspecttx_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/libs/proto"

	"github.com/nsf/jsondiff"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func TestMarshalTransactionAndInputDataToString(t *testing.T) {
	transaction := &commandspb.Transaction{}
	inputData := &commandspb.InputData{}

	marshalledTransaction, marshalledInputData, err := marshalTransactionAndInputDataToString(transaction, inputData)

	assert.NoErrorf(t, err, "Error marshalling transaction and input data: %v", err)
	assert.NotZero(t, len(marshalledTransaction), "Marshalled transaction is empty")
	assert.NotZero(t, len(marshalledInputData), "Marshalled input data is empty")
}

func TestUnmarshalTransactionReturnsValidTransactionAndInputData(t *testing.T) {
	transaction := &commandspb.Transaction{}

	encodedTransaction, err := proto.Marshal(transaction)
	assert.NoErrorf(t, err, "Error marshalling transaction: %v", err)

	unmarshalledTransaction, inputData, err := unmarshalTransaction(encodedTransaction)
	assert.NoErrorf(t, err, "Error unmarshalling transaction: %v", err)
	assert.NotNil(t, unmarshalledTransaction, "Unmarshalled transaction is nil")
	assert.NotNil(t, inputData, "Unmarshalled input data is nil")
}

type CompareJsonTestCase struct {
	Name         string
	FirstJson    []byte
	SecondJson   []byte
	ExpectedDiff jsondiff.Difference
}

func TestCompareJson(t *testing.T) {
	testCases := []CompareJsonTestCase{
		{
			Name:         "Same JSON",
			FirstJson:    []byte(`{"name": "John", "age": 30}`),
			SecondJson:   []byte(`{"name": "John", "age": 30}`),
			ExpectedDiff: jsondiff.FullMatch,
		},
		{
			Name:         "Different age",
			FirstJson:    []byte(`{"name": "John", "age": 30}`),
			SecondJson:   []byte(`{"name": "John", "age": 300}`),
			ExpectedDiff: jsondiff.NoMatch,
		},
		{
			Name:         "Different casing in name",
			FirstJson:    []byte(`{"name": "John", "age": 30}`),
			SecondJson:   []byte(`{"name": "john", "age": 300}`),
			ExpectedDiff: jsondiff.NoMatch,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			result, diffHtml := compareJson(testCase.FirstJson, testCase.SecondJson)

			assert.Equalf(t, testCase.ExpectedDiff, result, "Expected JSONs to have a difference of %v, but got: %v", testCase.ExpectedDiff, result)
			assert.NotZero(t, len(diffHtml), "Diff HTML is empty")
		})
	}
}
