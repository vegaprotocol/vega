package inspecttx_helpers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/jsonpb"
	"github.com/nsf/jsondiff"
	"github.com/spf13/cobra"
)

var (
	currentFile          string
	txDirectory          string
	diffOutputDirectory  string
	transactionDiffs     int
	transactionsAnalysed int
	transactionsPassed   int
	jsonMarshaller       = jsonpb.Marshaler{
		Indent: "   ",
	}
	inspectTxDirCmd = &cobra.Command{
		Use:   "inspect-tx-dir",
		Short: "inspect transactions in a given directory",
		Long:  "must be a directory containing only json files. The json files must have the structure defined in the README.md (if you are reading this and there is no readme then I have messed up, do not approve my PR",
		RunE:  inspectTxsInDirectoryCmd,
	}
)

func init() {
	rootCmd.AddCommand(inspectTxDirCmd)
	inspectTxDirCmd.Flags().StringVarP(&txDirectory, "txdir", "d", "", "directory containing json files with base64 encoded data and rawjson for a transaction")
	_ = inspectTxDirCmd.MarkFlagRequired("txdir")
	inspectTxDirCmd.Flags().StringVarP(&diffOutputDirectory, "diff-output-file", "o", "./transaction-diffs", "directory to output files containing transaction diffs to")
}

func inspectTxsInDirectoryCmd(_ *cobra.Command, _ []string) error {
	transactionsAnalysed = 0
	transactionsPassed = 0
	transactionDiffs = 0

	transactionFiles, err := getFilesInDirectory(txDirectory)
	if err != nil {
		return fmt.Errorf("error when attempting to get files in the given directory. \nerr: %v", err)
	}

	for _, file := range transactionFiles {
		currentFile = file
		transactionData, err := getTransactionDataFromFile(file)
		if err != nil {
			return fmt.Errorf("error reading transaction file '%s'\nerr: %v", file, err)
		}

		logrus.Infof("inspecting transactions in '%s'", file)
		err = inspectTransaction(transactionData)
		if err != nil {
			return fmt.Errorf("error when attempting to inspect transaction in file '%s' \nerr: %v", file, err)
		}
	}

	logrus.Infof("transactions analysed: %d, transactions passed: %d, transactions failed: %d", transactionsAnalysed, transactionsPassed, transactionDiffs)
	if transactionDiffs != 0 {
		return fmt.Errorf("there were diffs in the transactions sent from your application vs the marshalled equivalents from core, check your protos are up to date. Diffs can be found in '%s'\nnumber of transactions with diffs: %d", diffOutputDirectory, transactionDiffs)
	}

	return nil
}

func inspectTransaction(transactionData TransactionData) error {
	decodedBytes, err := base64.StdEncoding.DecodeString(transactionData.EncodedData)
	if err != nil {
		return fmt.Errorf("error occurred when decoding the transaction encoded data.\nerr: %v", err)
	}

	unmarshalledTransaction, unmarshalledInputData, err := unmarshalTransaction(decodedBytes)
	if err != nil {
		return fmt.Errorf("an error occurred when attempting to unmarshal the decoded transaction byte array. \nerr: %v", err)
	}

	marshalledTransaction, marshalledInputData, err := marshalTransactionAndInputDataToString(unmarshalledTransaction, unmarshalledInputData)
	if err != nil {
		return fmt.Errorf("an error occurred when attempting to marshal the structs back to a json string. \nerr: %v", err)
	}

	transactionCompareResult, transactionDiffHtml := compareJson(transactionData.Transaction, []byte(marshalledTransaction))
	inputDataCompareResult, inputDataDiffHtml := compareJson(transactionData.InputData, []byte(marshalledInputData))

	if transactionCompareResult == jsondiff.NoMatch || inputDataCompareResult == jsondiff.NoMatch {
		transactionDiffs += 1

		if transactionCompareResult == jsondiff.NoMatch {
			comparableTransactionJson := ComparableJson{
				OriginalJson: transactionData.Transaction,
				CoreJson:     json.RawMessage(marshalledTransaction),
				DiffType:     Transaction,
			}

			logrus.Errorf("transaction data did not match, writing diff data to %s", diffOutputDirectory)
			err = writeDiffToFile(comparableTransactionJson, transactionDiffHtml)
			if err != nil {
				return fmt.Errorf("error occurred when attempting to write transaction diffs to file. \nerr: %v", err)
			}
		}

		if inputDataCompareResult == jsondiff.NoMatch {
			comparableInputDataJson := ComparableJson{
				OriginalJson: transactionData.InputData,
				CoreJson:     json.RawMessage(marshalledInputData),
				DiffType:     InputData,
			}

			logrus.Errorf("input data did not match, writing diff data to %s", diffOutputDirectory)
			err = writeDiffToFile(comparableInputDataJson, inputDataDiffHtml)
			if err != nil {
				return fmt.Errorf("error occurred when attempting to write input data diffs to file. \nerr: %v", err)
			}
		}
	} else {
		transactionsPassed += 1
	}

	transactionsAnalysed += 1

	return nil
}
