package tools

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	"code.vegaprotocol.io/vega/libs/proto"

	"code.vegaprotocol.io/vega/commands"

	inspecttx_helpers "code.vegaprotocol.io/vega/vegatools/inspecttx/inspecttx-helpers"
	"github.com/stretchr/testify/assert"
)

const (
	testFilesValid   = "./testfiles/valid"
	testFilesInvalid = "./testfiles/invalid"
)

func TestInspectTxsInDirectoryCmd_ReturnsNoErrorWhenAllTransactionsMatch(t *testing.T) {
	testTransaction := commands.NewTransaction("mykey", nil, commands.NewSignature([]byte("bob"), "dave", 3))
	testTransactionAlias := &inspecttx_helpers.TransactionAlias{Transaction: testTransaction}
	transactionJson, err := testTransactionAlias.MarshalJSON()
	data := inspecttx_helpers.TransactionData{Transaction: json.RawMessage(transactionJson)}

	fmt.Print(string(transactionJson))
	marshalledProto, err := proto.Marshal(testTransaction)
	encodedData := base64.StdEncoding.EncodeToString(marshalledProto)
	data.EncodedData = encodedData

	os.MkdirAll(testFilesValid, 0o755)
	filePath := path.Join(testFilesValid, "transaction.json")
	jsonData, err := json.Marshal(data)
	err = os.WriteFile(filePath, jsonData, 0o644)

	cmd := checkTxCmd{
		txDirectory:   testFilesValid,
		diffOutputDir: testFilesValid + "/diffs",
	}

	err = cmd.Execute(nil)
	assert.NoErrorf(t, err, "expected inspectTxsInDirectoryCmd to run without error, however one was thrown: \nERR: %v", err)

	files, err := inspecttx_helpers.GetFilesInDirectory(cmd.diffOutputDir)
	assert.NoErrorf(t, err, "error occurred when checking diff dir for files\nerr: %v", err)
	assert.Len(t, files, 0, "expected to find no diff files for passing transactions, but files were located")

	err = os.RemoveAll(cmd.diffOutputDir)
	assert.Error(t, err, "error cleaning up diff dir")
	err = os.RemoveAll(testFilesValid)
	assert.Error(t, err, "error cleaning up diff dir")
}

func TestInspectTxsInDirectoryCmd_ErrReturnedAfterInspectingAllFilesIfNoMatch(t *testing.T) {
	cmd := checkTxCmd{
		txDirectory:   testFilesInvalid,
		diffOutputDir: testFilesInvalid + "/diffs",
	}

	err := cmd.Execute(nil)
	assert.Error(t, err, "inspectTxsInDirectoryCmd was expected to fail, however the command did not throw any error")
	err = os.RemoveAll(cmd.diffOutputDir)
	assert.NoError(t, err, "error cleaning up diff dir")
}
