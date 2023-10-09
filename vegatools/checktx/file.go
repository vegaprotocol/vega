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
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GetFilesInDirectory(directory string) ([]string, error) {
	files, err := os.Open(directory)
	if err != nil {
		return nil, fmt.Errorf("error occurred when attempting to open the given directory '%s'. \nerr: %w", directory, err)
	}
	defer func(files *os.File) {
		err := files.Close()
		if err != nil {
			panic(err)
		}
	}(files)

	fileInfo, err := files.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("an error occurred when attempting to read files in the given directory. \nerr: %w", err)
	}

	transactionFiles := make([]string, 0, len(fileInfo))
	for _, info := range fileInfo {
		dir := filepath.Join(directory, info.Name())
		transactionFiles = append(transactionFiles, dir)
	}

	return transactionFiles, nil
}

func GetEncodedTransactionFromFile(filePath string) (string, error) {
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file at %s. \nerr: %w", filePath, err)
	}

	_, err = base64.StdEncoding.DecodeString(string(fileContents))
	if err != nil {
		return "", fmt.Errorf("error occurred when attempting to decode transaction data in %s, is there  definitely base64 encoded data in your file?\nerr: %v", filePath, err)
	}

	data := strings.TrimSpace(string(fileContents))
	return data, nil
}
