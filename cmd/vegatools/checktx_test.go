package tools

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path"
	"testing"

	"code.vegaprotocol.io/vega/vegatools/inspecttx"

	"code.vegaprotocol.io/vega/libs/proto"

	v1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"code.vegaprotocol.io/vega/commands"

	"github.com/stretchr/testify/assert"
)

const (
	testFiles = "./testdata/transactionfiles"
	diffFiles = "./testdata/diffs"
)

var testInputData = v1.InputData{Nonce: 123, BlockHeight: 456, Command: &v1.InputData_Transfer{Transfer: &v1.Transfer{
	FromAccountType: 1,
	To:              "dave",
	ToAccountType:   2,
	Asset:           "test asset",
	Amount:          "123",
	Reference:       "test ref",
	Kind:            nil,
}}}

func setUpTestData(t *testing.T) inspecttx.TransactionData {
	marshalledInputData, err := commands.MarshalInputData(&testInputData)
	assert.NoErrorf(t, err, "error occurred when attempting to marshal test input data\nerr: %v", err)
	testTransaction := commands.NewTransaction("mykey", marshalledInputData, commands.NewSignature([]byte("testSig"), "testAlgo", 3))

	transactionJson, err := inspecttx.MarshalToJSONWithOneOf(testTransaction)
	assert.NoErrorf(t, err, "error occurred when attempting to marshal raw transaction to json: %v", err)
	inputDataJson, err := inspecttx.MarshalToJSONWithOneOf(&testInputData)
	assert.NoErrorf(t, err, "error occurred when attempting to marshal raw input data to json: %v", err)
	protoToEncode, err := proto.Marshal(testTransaction)
	assert.NoErrorf(t, err, "error occurred when attempting to encode the transaction proto %v", err)
	encodedData := base64.StdEncoding.EncodeToString(protoToEncode)

	testData := inspecttx.TransactionData{Transaction: json.RawMessage(transactionJson), InputData: json.RawMessage(inputDataJson), EncodedData: encodedData}
	return testData
}

func writeTestDataToFile(t *testing.T, testData inspecttx.TransactionData, testFilePath string, testfileName string) {
	transactionData, err := json.Marshal(testData)
	err = os.MkdirAll(testFilePath, 0o755)
	assert.NoErrorf(t, err, "error occurred when attempting to make a directory for the valid test data")
	filePath := path.Join(testFilePath, testfileName)

	err = os.WriteFile(filePath, transactionData, 0o644)
	assert.NoErrorf(t, err, "error when creating transaction.json file.\nerr: %v", err)

	err = os.MkdirAll(diffFiles, 0o755)
	assert.NoErrorf(t, err, "error occurred when attempting to make a directory for test diffs")
}

func clearTestData(t *testing.T) {
	err := os.RemoveAll(testFiles)
	assert.NoErrorf(t, err, "error occurred when attempting to clean valid test data dir")
}

func TestInspectTxsInDirectoryCmd_ReturnsNoErrorWhenAllTransactionsMatch(t *testing.T) {
	clearTestData(t)
	writeTestDataToFile(t, setUpTestData(t), testFiles, "transaction1.json")
	writeTestDataToFile(t, setUpTestData(t), testFiles, "transaction2.json")
	cmd := checkTxCmd{
		Transactions: testFiles,
		Diffs:        diffFiles,
	}

	err := cmd.Execute(nil)
	assert.NoErrorf(t, err, "expected inspectTxsInDirectoryCmd to run without error, however one was thrown: \nERR: %v", err)
	assert.Equalf(t, 0, transactionDiffs, "expected there to be no comparison failures, instead there was %d", transactionDiffs)
	assert.Equalf(t, 2, transactionsPassed, "expected there to be 1 passing transaction, instead there was %d", transactionsPassed)
	assert.Equalf(t, 2, transactionsAnalysed, "expected there to be 1 analysed transaction, instead there was %d", transactionsAnalysed)
}

func TestInspectTxsInDirectoryCmd_ErrReturnedAfterInspectingAllFilesIfNoMatch(t *testing.T) {
	clearTestData(t)
	data := setUpTestData(t)
	var jsonMap map[string]interface{}
	err := json.Unmarshal(data.Transaction, &jsonMap)

	jsonMap["version"] = "this will cause a diff"
	jsonForCausingDiff, err := json.Marshal(jsonMap)
	data.Transaction = jsonForCausingDiff

	writeTestDataToFile(t, data, testFiles, "transaction.json")
	writeTestDataToFile(t, setUpTestData(t), testFiles, "transaction1.json")
	cmd := checkTxCmd{
		Transactions: testFiles,
		Diffs:        diffFiles,
	}

	err = cmd.Execute(nil)
	assert.Error(t, err, "inspectTxsInDirectoryCmd was expected to fail, however the command did not throw any error")
	assert.Equalf(t, 1, transactionDiffs, "expected there to be 1 failure, instead there was %d", transactionDiffs)
	assert.Equalf(t, 1, transactionsPassed, "expected there to be no passing transactions, instead there was %d", transactionsPassed)
	assert.Equalf(t, 2, transactionsAnalysed, "expected there to be 1 analysed transaction, instead there was %d", transactionsAnalysed)
}
