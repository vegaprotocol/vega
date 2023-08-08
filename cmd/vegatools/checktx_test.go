package tools

import (
	"encoding/base64"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/proto"
	v1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

const txVersion = 3

var testInputData = v1.InputData{Nonce: 123, BlockHeight: 456, Command: &v1.InputData_Transfer{Transfer: &v1.Transfer{
	FromAccountType: 1,
	To:              "dave",
	ToAccountType:   2,
	Asset:           "test asset",
	Amount:          "123",
	Reference:       "test ref",
	Kind:            nil,
}}}

func TestTxReturnsNoErrorWhenCheckingCompatibleTransaction(t *testing.T) {
	encodedTransaction := createdEncodedTransactionData(t)
	cmd := checkTxCmd{
		EncodedTransaction: encodedTransaction,
	}

	err := cmd.Execute(nil)
	assert.NoErrorf(t, err, "error was returned when the transaction should have been valid")
}

func TestTxReturnsErrorWhenCheckingIncompatibleTransaction(t *testing.T) {
	cmd := checkTxCmd{
		EncodedTransaction: "12345",
	}

	err := cmd.Execute(nil)
	assert.Error(t, err, "")
}

func TestMarshalAndEncodeTransaction(t *testing.T) {
	encoded, err := marshalAndEncodeTransaction(&v1.Transaction{})
	assert.NoError(t, err)
	assertIsBase64Encoded(t, encoded)
}

func TestDecodeAndUnmarshalTransactionUnmarshalsEncodedData(t *testing.T) {
	unmarshalled, err := decodeAndUnmarshalTransaction(createdEncodedTransactionData(t))
	assert.NoError(t, err)
	assert.Equalf(t, v1.TxVersion(txVersion), unmarshalled.Version, "expected version to be set in the unmarshalled data")
}

func TestDecodeAndUnmarshalTransactionThrowsErrorWithInvalidData(t *testing.T) {
	_, err := decodeAndUnmarshalTransaction("invalid")
	assert.Error(t, err)
}

func createdEncodedTransactionData(t *testing.T) string {
	t.Helper()
	marshalledInputData, err := commands.MarshalInputData(&testInputData)
	assert.NoErrorf(t, err, "error occurred when mashalling test input data")
	transaction := commands.NewTransaction("mypubkey", marshalledInputData, commands.NewSignature([]byte("sig"), "dummyalgo", txVersion))
	transactionProto, err := proto.Marshal(transaction)
	assert.NoErrorf(t, err, "error occurred when marshalling the test data to a proto")
	encodedTransaction := base64.StdEncoding.EncodeToString(transactionProto)
	return encodedTransaction
}

func assertIsBase64Encoded(t *testing.T, encodedStr string) {
	t.Helper()
	_, err := base64.StdEncoding.DecodeString(encodedStr)
	assert.NoError(t, err, "Expected the string to be base64 encoded")
}
