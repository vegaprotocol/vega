package checktx

import (
	"encoding/base64"
	"os"
	"path"
	"testing"

	"github.com/golang/protobuf/jsonpb"

	v1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

const (
	txVersion   = 3
	testFileDir = "./testfiles"
)

func createTestDataFile(t *testing.T, fileName string, encodedTransaction string) {
	t.Helper()
	err := os.MkdirAll(testFileDir, 0o755)
	assert.NoErrorf(t, err, "error occurred when attempting to make a directory for the valid test data")

	filePath := path.Join(testFileDir, fileName)

	err = os.WriteFile(filePath, []byte(encodedTransaction), 0o644)
	assert.NoErrorf(t, err, "error when creating transaction.json file.\nerr: %v", err)
}

func clearTestData(t *testing.T) {
	t.Helper()
	err := os.RemoveAll(testFileDir)
	assert.NoErrorf(t, err, "error occurred when attempting to clean valid test data dir")
}

func TestCheckTransactionsInDirectoryThrowsNoErrAndReturnsAccurateMetrics(t *testing.T) {
	defer clearTestData(t)
	encodedTransaction, err := CreatedEncodedTransactionData()
	assert.NoErrorf(t, err, "error occurred when attempting to create encoded test data")

	createTestDataFile(t, "transaction1.txt", encodedTransaction)
	createTestDataFile(t, "transaction2.txt", encodedTransaction)

	resultData, err := CheckTransactionsInDirectory(testFileDir)
	assert.NoErrorf(t, err, "expected no error to occur when analysing valid transactions. Err: %v", err)
	assert.Equalf(t, 2, resultData.TransactionsAnalysed, "expected 2 transactions to have been analysed, instead there was %d", resultData.TransactionsAnalysed)
	assert.Equalf(t, 2, resultData.TransactionsPassed, "expected 2 transactions to have passed, instead there was %d", resultData.TransactionsPassed)
	assert.Equalf(t, 0, resultData.TransactionsFailed, "expected 0 transactions to have failed, instead there was %d", resultData.TransactionsFailed)
}

func TestCheckTransactionsInDirectoryThrowsErrIfFileContainsInvalidBase64Data(t *testing.T) {
	defer clearTestData(t)
	encodedTransaction, err := CreatedEncodedTransactionData()
	assert.NoErrorf(t, err, "error occurred when attempting to create encoded test data")

	createTestDataFile(t, "transaction1.txt", "12345")
	createTestDataFile(t, "transaction2.txt", encodedTransaction)

	resultData, err := CheckTransactionsInDirectory(testFileDir)
	assert.Errorf(t, err, "expected to exit CheckTransactionsInDirectory with an err when one of the files has invalid data, no error was thrown")
	assert.Equalf(t, 0, resultData.TransactionsAnalysed, "expected 0 transactions to have been analysed, instead there was %d", resultData.TransactionsAnalysed)
	assert.Equalf(t, 0, resultData.TransactionsPassed, "expected 0 transactions to have passed, instead there was %d", resultData.TransactionsPassed)
	assert.Equalf(t, 0, resultData.TransactionsFailed, "expected 0 transactions to have failed, instead there was %d", resultData.TransactionsFailed)
}

func TestCheckTransactionsInDirectoryAccuratelyReportsFailures(t *testing.T) {
	defer clearTestData(t)
	encodedTransaction, err := CreatedEncodedTransactionData()
	assert.NoErrorf(t, err, "error occurred when attempting to create encoded test data")

	transactionForFailScenario, err := CreateTransaction()
	assert.NoErrorf(t, err, "error occurred when attempting to create encoded test data")

	marshaller := jsonpb.Marshaler{
		OrigName:     false,
		EnumsAsInts:  false,
		EmitDefaults: false,
		Indent:       "",
		AnyResolver:  nil,
	}

	failScenarioJson, err := marshaller.MarshalToString(transactionForFailScenario)
	assert.NoErrorf(t, err, "error occurred when attempting to marshal transaction json to string. Err: %v", err)
	failScenarioNonProtoEncode := base64.StdEncoding.EncodeToString([]byte(failScenarioJson))

	createTestDataFile(t, "transaction1.txt", encodedTransaction)
	createTestDataFile(t, "transaction2.txt", failScenarioNonProtoEncode)

	resultData, err := CheckTransactionsInDirectory(testFileDir)
	assert.NoErrorf(t, err, "expected no error from CheckTransactionsInDirectory when analysing valid base64 encoded data. Err: %v", err)
	assert.Equalf(t, 2, resultData.TransactionsAnalysed, "expected 2 transactions to have been analysed, instead there was %d", resultData.TransactionsAnalysed)
	assert.Equalf(t, 1, resultData.TransactionsPassed, "expected 1 transactions to have passed, instead there was %d", resultData.TransactionsPassed)
	assert.Equalf(t, 1, resultData.TransactionsFailed, "expected 1 transactions to have failed, instead there was %d", resultData.TransactionsFailed)
}

func TestMarshalAndEncodeTransaction(t *testing.T) {
	encoded, err := marshalAndEncodeTransaction(&v1.Transaction{})
	assert.NoError(t, err)
	assertIsBase64Encoded(t, encoded)
}

func TestDecodeAndUnmarshalTransactionUnmarshalsEncodedData(t *testing.T) {
	encodedTransaction, err := CreatedEncodedTransactionData()
	assert.NoError(t, err)

	unmarshalled, err := decodeAndUnmarshalTransaction(encodedTransaction)
	assert.NoError(t, err)
	assert.Equalf(t, v1.TxVersion(txVersion), unmarshalled.Version, "expected version to be set in the unmarshalled data")
	assert.Equalf(t, TestAlgoName, unmarshalled.Signature.Algo, "algo to be set in the unmarshalled data")
}

func TestDecodeAndUnmarshalTransactionThrowsErrorWithInvalidData(t *testing.T) {
	_, err := decodeAndUnmarshalTransaction("invalid")
	assert.Error(t, err)
}

func assertIsBase64Encoded(t *testing.T, encodedStr string) {
	t.Helper()
	_, err := base64.StdEncoding.DecodeString(encodedStr)
	assert.NoError(t, err, "Expected the string to be base64 encoded")
}
