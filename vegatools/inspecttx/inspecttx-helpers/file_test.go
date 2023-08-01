package inspecttx_helpers

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestGetFilesInDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	txDirectory = tmpDir

	testFiles := []string{"file1.json", "file2.json", "file3.json"}
	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		if _, err := os.Create(filePath); err != nil {
			t.Fatalf("failed to create test file %s: %v", filePath, err)
		}
	}

	files, err := getFilesInDirectory()
	if err != nil {
		t.Fatalf("getFilesInDirectory failed: %v", err)
	}

	// Check if all test files are present in the returned list
	if len(files) != len(testFiles) {
		t.Fatalf("getFilesInDirectory returned incorrect number of files: got %d, expected %d", len(files), len(testFiles))
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		found := false
		for _, f := range files {
			if f == filePath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("getFilesInDirectory did not return expected file: %s", filePath)
		}
	}
}

func TestReadTransactionFile(t *testing.T) {
	// Create a temporary test file
	tmpFile, err := ioutil.TempFile("", "test-transaction-*.json")
	if err != nil {
		t.Fatalf("failed to create temporary test file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testData := `{"Transaction": {"field1": "value1"}, "InputData": {"field2": "value2"}, "EncodedData": "base64data"}`
	if _, err := tmpFile.Write([]byte(testData)); err != nil {
		t.Fatalf("failed to write test data to file: %v", err)
	}
	tmpFile.Close()

	transactionData, err := readTransactionFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("readTransactionFile failed: %v", err)
	}

	// Check if the data was correctly unmarshalled
	var expectedData TransactionData
	err = json.Unmarshal([]byte(testData), &expectedData)
	if err != nil {
		t.Fatalf("failed to unmarshal expected data: %v", err)
	}

	if string(transactionData.Transaction) != string(expectedData.Transaction) {
		t.Errorf("readTransactionFile returned incorrect Transaction data: got %s, expected %s", transactionData.Transaction, expectedData.Transaction)
	}

	if string(transactionData.InputData) != string(expectedData.InputData) {
		t.Errorf("readTransactionFile returned incorrect InputData: got %s, expected %s", transactionData.InputData, expectedData.InputData)
	}

	if transactionData.EncodedData != expectedData.EncodedData {
		t.Errorf("readTransactionFile returned incorrect EncodedData: got %s, expected %s", transactionData.EncodedData, expectedData.EncodedData)
	}
}

func TestTrimExtensionFromCurrentFileName(t *testing.T) {
	currentFile = "/path/to/somefile.json"
	trimmedFileName := trimExtensionFromCurrentFileName()

	expectedTrimmedFileName := "somefile"
	if trimmedFileName != expectedTrimmedFileName {
		t.Errorf("trimExtensionFromCurrentFileName returned incorrect trimmed file name: got %s, expected %s", trimmedFileName, expectedTrimmedFileName)
	}
}
