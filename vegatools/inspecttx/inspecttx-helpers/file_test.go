package inspecttx_helpers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFilesInDirectoryRetrievesAllFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFiles := []string{"file1.json", "file2.json", "file3.json"}
	filePaths := make([]string, 0, len(testFiles))

	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		if _, err := os.Create(filePath); err != nil {
			t.Fatalf("failed to create test file %s: %v", filePath, err)
		}
		filePaths = append(filePaths, filePath)
	}

	retrievedFiles, err := GetFilesInDirectory(tmpDir)
	assert.NoError(t, err, "GetFilesInDirectory failed")
	assert.Len(t, retrievedFiles, len(testFiles), "GetFilesInDirectory returned incorrect number of files")

	for _, filePath := range filePaths {
		assert.Contains(t, retrievedFiles, filePath, "GetFilesInDirectory did not return expected file")
	}
}

func TestGetTransactionDataFromFileReturnsValidTransactionData(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-transaction-*.json")
	assert.NoErrorf(t, err, "failed to create temporary test file: %v", err)
	defer os.Remove(tmpFile.Name())

	testData := `{"Transaction": {"field1": "value1"}, "InputData": {"field2": "value2"}, "EncodedData": "base64data"}`
	_, err = tmpFile.WriteString(testData)
	assert.NoErrorf(t, err, "failed to write test data to file: %v", err)
	err = tmpFile.Close()
	assert.NoErrorf(t, err, "error occurred when closing file: %v", err)

	transactionData, err := GetTransactionDataFromFile(tmpFile.Name())
	assert.NoError(t, err, "GetTransactionDataFromFile failed")

	var expectedData TransactionData
	err = json.Unmarshal([]byte(testData), &expectedData)
	assert.NoError(t, err, "failed to unmarshal expected data")

	assert.Equal(t, string(expectedData.Transaction), string(transactionData.Transaction), "GetTransactionDataFromFile returned incorrect Transaction data")
	assert.Equal(t, string(expectedData.InputData), string(transactionData.InputData), "GetTransactionDataFromFile returned incorrect InputData")
	assert.Equal(t, expectedData.EncodedData, transactionData.EncodedData, "GetTransactionDataFromFile returned incorrect EncodedData")
}

func TestTrimExtensionFromCurrentFileName(t *testing.T) {
	file := "/path/to/somefile.json"
	trimmedFileName := trimExtensionFromFileName(file)

	expectedTrimmedFileName := "somefile"
	assert.Equal(t, expectedTrimmedFileName, trimmedFileName, "trimExtensionFromCurrentFileName returned incorrect trimmed file name")
}
