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
