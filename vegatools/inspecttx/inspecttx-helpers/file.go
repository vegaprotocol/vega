package inspecttx_helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

type TransactionData struct {
	Transaction json.RawMessage
	InputData   json.RawMessage
	EncodedData string
}

type ComparableJson struct {
	OriginalJson json.RawMessage
	CoreJson     json.RawMessage
	DiffType     DiffType
}

type DiffType string

const (
	InputData   DiffType = "InputData"
	Transaction DiffType = "Transaction"
)

func getFilesInDirectory() ([]string, error) {
	files, err := os.Open(txDirectory)
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

	var transactionFiles []string
	for _, info := range fileInfo {
		dir := filepath.Join(txDirectory, info.Name())
		transactionFiles = append(transactionFiles, dir)
	}

	return transactionFiles, nil
}

func readTransactionFile(filePath string) (TransactionData, error) {
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

func writeDiffToFile(diffData ComparableJson, html string) {
	marshalledDiffData, err := json.MarshalIndent(diffData, " ", "	")
	if err != nil {
		logrus.Warnf("error marshalling diffs to json when preparing to write diffs to file. \nerr: %v", err)
	}

	folderName := trimExtensionFromCurrentFileName()
	filePath := path.Join(diffOutputDirectory, folderName)

	if err := os.MkdirAll(filePath, 0o755); err != nil {
		logrus.Warn("Error creating directory:", err)
		return
	}

	jsonFileName := fmt.Sprintf("%s-tocompare.json", string(diffData.DiffType))
	err = os.WriteFile(path.Join(filePath, jsonFileName), marshalledDiffData, 0o644)
	if err != nil {
		logrus.Warnf("error when attempting to write diffs to file.\nerr: %v", err)
	}

	htmlFileName := fmt.Sprintf("%s-diff.html", string(diffData.DiffType))
	err = os.WriteFile(path.Join(filePath, htmlFileName), []byte(html), 0o644)
	if err != nil {
		logrus.Warnf("error when attempting to write diffs to html file.\nerr: %v", err)
	}
}
