package inspecttx_helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type TransactionData struct {
	Transaction json.RawMessage
	InputData   json.RawMessage
	EncodedData string
}

type DiffType string

const (
	InputData   DiffType = "InputData"
	Transaction DiffType = "Transaction"
)

func getFilesInDirectory(directory string) ([]string, error) {
	files, err := os.Open(directory)
	if err != nil {
		return nil, fmt.Errorf("error occurred when attempting to open the given directory. \nerr: %w", err)
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

func getTransactionDataFromFile(filePath string) (TransactionData, error) {
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return TransactionData{}, fmt.Errorf("error reading file at %s. \nerr: %w", filePath, err)
	}

	transactionData := TransactionData{}
	err = json.Unmarshal(fileContents, &transactionData)
	if err != nil {
		return TransactionData{}, fmt.Errorf("error unmarshalling the json in file '%s'. \nerr: %v", filePath, err)
	}

	return transactionData, nil
}

func trimExtensionFromCurrentFileName() string {
	dotIndex := strings.Index(currentFile, ".")
	lastSlashIndex := strings.LastIndex(currentFile, "/")
	trimmedFileName := currentFile
	if dotIndex != -1 {
		trimmedFileName = currentFile[lastSlashIndex+1 : dotIndex]
	}

	return trimmedFileName
}

func writeToFile(filePath string, data []byte, fileMode os.FileMode) error {
	err := os.WriteFile(filePath, data, fileMode)
	if err != nil {
		return fmt.Errorf("error writing file '%s': %w", filePath, err)
	}
	return nil
}

func writeDiffToFile(diffData ComparableJson, html string) error {
	marshalledDiffData, err := json.MarshalIndent(diffData, " ", "	")
	if err != nil {
		return fmt.Errorf("error marshalling diffs to json when preparing to write diffs to file. \nerr: %v", err)
	}

	folderName := trimExtensionFromCurrentFileName()
	filePath := path.Join(diffOutputDirectory, folderName)

	if err := os.MkdirAll(filePath, 0o755); err != nil {
		return fmt.Errorf("error creating directory for diff files. \nerr: %v", err)
	}

	jsonFileName := fmt.Sprintf("%s-tocompare.json", string(diffData.DiffType))
	if err := writeToFile(path.Join(filePath, jsonFileName), marshalledDiffData, 0o644); err != nil {
		return fmt.Errorf("error when attempting to write diffs to JSON file.\nerr: %v", err)
	}

	htmlFileName := fmt.Sprintf("%s-diff.html", string(diffData.DiffType))
	if err := writeToFile(path.Join(filePath, htmlFileName), []byte(html), 0o644); err != nil {
		return fmt.Errorf("error when attempting to write diffs to HTML file.\nerr: %v", err)
	}

	return nil
}
