package inspecttx_helpers

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFilesInDirectoryRetrievesAllFiles(t *testing.T) {
	tmpDir := t.TempDir()

	var filePaths []string
	testFiles := []string{"file1.json", "file2.json", "file3.json"}
	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		if _, err := os.Create(filePath); err != nil {
			t.Fatalf("failed to create test file %s: %v", filePath, err)
		}
		filePaths = append(filePaths, filePath)
	}

	retrievedFiles, err := getFilesInDirectory(tmpDir)
	assert.NoError(t, err, "getFilesInDirectory failed")
	assert.Len(t, retrievedFiles, len(testFiles), "getFilesInDirectory returned incorrect number of files")

	for _, filePath := range filePaths {
		assert.Contains(t, retrievedFiles, filePath, "getFilesInDirectory did not return expected file")
	}
}

func TestGetTransactionDataFromFileReturnsValidTransactionData(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "test-transaction-*.json")
	assert.NoErrorf(t, err, "failed to create temporary test file: %v", err)
	defer os.Remove(tmpFile.Name())

	testData := `{"Transaction": {"field1": "value1"}, "InputData": {"field2": "value2"}, "EncodedData": "base64data"}`
	_, err = tmpFile.WriteString(testData)
	assert.NoErrorf(t, err, "failed to write test data to file: %v", err)
	tmpFile.Close()

	transactionData, err := getTransactionDataFromFile(tmpFile.Name())
	assert.NoError(t, err, "getTransactionDataFromFile failed")

	var expectedData TransactionData
	err = json.Unmarshal([]byte(testData), &expectedData)
	assert.NoError(t, err, "failed to unmarshal expected data")

	assert.Equal(t, string(expectedData.Transaction), string(transactionData.Transaction), "getTransactionDataFromFile returned incorrect Transaction data")
	assert.Equal(t, string(expectedData.InputData), string(transactionData.InputData), "getTransactionDataFromFile returned incorrect InputData")
	assert.Equal(t, expectedData.EncodedData, transactionData.EncodedData, "getTransactionDataFromFile returned incorrect EncodedData")
}

func TestTrimExtensionFromCurrentFileName(t *testing.T) {
	currentFile = "/path/to/somefile.json"
	trimmedFileName := trimExtensionFromCurrentFileName()

	expectedTrimmedFileName := "somefile"
	assert.Equal(t, expectedTrimmedFileName, trimmedFileName, "trimExtensionFromCurrentFileName returned incorrect trimmed file name")
}
