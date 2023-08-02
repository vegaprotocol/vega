package inspecttx_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testFilesValid   = "./testfiles/valid"
	testFilesInvalid = "./testfiles/invalid"
)

func TestInspectTxsInDirectoryCmd_ReturnsNoErrorWhenAllTransactionsMatch(t *testing.T) {
	txDirectory = testFilesValid
	files, err := getFilesInDirectory(txDirectory)
	assert.NoErrorf(t, err, "expected to be able to retrieve files from test directory")
	numFiles := len(files)
	err = inspectTxsInDirectoryCmd(nil, nil)
	assert.NoErrorf(t, err, "expected inspectTxsInDirectoryCmd to run without error, however one was thrown: \nERR: %v", err)
	assert.Equal(t, numFiles, transactionsAnalysed)
	assert.Equal(t, 0, transactionDiffs)
	assert.Equal(t, 2, transactionsPassed)
}

func TestInspectTxsInDirectoryCmd_ErrReturnedAfterInspectingAllFilesIfNoMatch(t *testing.T) {
	txDirectory = testFilesInvalid
	files, err := getFilesInDirectory(txDirectory)
	assert.NoErrorf(t, err, "expected to be able to retrieve files from test directory")
	numFiles := len(files)

	err = inspectTxsInDirectoryCmd(nil, nil)
	assert.Error(t, err, "inspectTxsInDirectoryCmd was expected to fail, however the command did not throw any error")
	assert.Equal(t, numFiles, transactionsAnalysed)
	assert.Equal(t, 2, transactionDiffs)
	assert.Equal(t, 0, transactionsPassed)
}
