// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package checktx

import (
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

func TestGetEncodedTransactionFromFileReturnsBase64DataWhenBase64DataWritten(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-transaction-*.txt")
	assert.NoErrorf(t, err, "failed to create temporary test file: %v", err)
	defer os.Remove(tmpFile.Name())

	testData, err := CreatedEncodedTransactionData()
	assert.NoErrorf(t, err, "failed to create temporary test file: %v", err)

	_, err = tmpFile.WriteString(testData)
	assert.NoErrorf(t, err, "failed to write test data to file: %v", err)
	err = tmpFile.Close()
	assert.NoErrorf(t, err, "error occurred when closing file: %v", err)

	transactionData, err := GetEncodedTransactionFromFile(tmpFile.Name())
	assert.NoError(t, err, "GetTransactionDataFromFile failed")

	assertIsBase64Encoded(t, transactionData)
	assert.NoError(t, err, "failed to unmarshal expected data")
}
