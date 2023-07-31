package inspecttx_helpers

import (
	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/nsf/jsondiff"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var (
	txDirectory      string
	transactionDiffs = 0
	inputDataDiffs   = 0
	jsonMarshaller   = jsonpb.Marshaler{
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
	inspectTxDirCmd.MarkFlagRequired("txdir")
}

type TransactionData struct {
	Transaction json.RawMessage
	EncodedData string
}

func getFilesInDirectory() ([]string, error) {
	files, err := os.Open(txDirectory)
	if err != nil {
		return nil, fmt.Errorf("error occurred when attempting to open the given directory. \nerr: %w", err)
	}
	defer files.Close()

	fileInfo, err := files.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("an error occured when attempting to read files in the given directory. \nerr: %w", err)
	}

	var transactionFiles []string
	for _, info := range fileInfo {
		dir := filepath.Join(txDirectory, info.Name())
		transactionFiles = append(transactionFiles, dir)
	}

	return transactionFiles, nil
}

func inspectTxsInDirectoryCmd(cmd *cobra.Command, args []string) error {
	transactionFiles, err := getFilesInDirectory()
	if err != nil {
		return fmt.Errorf("error when attempting to get files in the given directory. \nerr: %w", err)
	}

	for _, file := range transactionFiles {
		fileContents, err := os.ReadFile(file)
		if err != nil {
			fmt.Sprintf("error reading file")
		}
		transactionData := TransactionData{}

		json.Unmarshal(fileContents, &transactionData)
		fmt.Sprintf(string(transactionData.Transaction))
		runInspectTx(transactionData)
	}

	if transactionDiffs != 0 {
		return fmt.Errorf("there were diffs in the transactions sent from your application vs the marshalled equivalents from core, check your protos are up to date\nnumber of transactions with diffs: %d", transactionDiffs)
	}

	return nil
}

func compareJson(firstJson []byte, secondJson []byte) (jsondiff.Difference, string) {
	options := jsondiff.DefaultConsoleOptions()
	result, diffString := jsondiff.Compare(firstJson, secondJson, &options)

	return result, diffString
}

func getUnmarshalledTransactionAndInputData(decodedTransactionByes []byte) (*commandspb.Transaction, *commandspb.InputData, error) {
	var unmarshalledTransaction = &commandspb.Transaction{}
	var unmarshalledInputData = &commandspb.InputData{}
	if err := proto.Unmarshal(decodedTransactionByes, unmarshalledTransaction); err != nil {
		return unmarshalledTransaction, unmarshalledInputData, fmt.Errorf("unable to unmarshal transaction. \nerr: %w", err)
	}

	unmarshalledInputData, err := commands.UnmarshalInputData(unmarshalledTransaction.InputData)
	if err != nil {
		return unmarshalledTransaction, unmarshalledInputData, fmt.Errorf("unable to unmarshal input data. \nerr: %w", err)
	}

	return unmarshalledTransaction, unmarshalledInputData, nil
}

func marshalTransactionAndInputDataToString(transaction *commandspb.Transaction, inputData *commandspb.InputData) (string, string, error) {
	marshalledTransaction, err := jsonMarshaller.MarshalToString(transaction)
	if err != nil {
		return "", "", fmt.Errorf("couldn't marshal transaction: %w", err)
	}

	marshalledInputData, err := jsonMarshaller.MarshalToString(inputData)
	if err != nil {
		return "", "", fmt.Errorf("couldn't marshal input data: %w", err)
	}

	return marshalledTransaction, marshalledInputData, nil
}

func runInspectTx(transactionData TransactionData) error {
	decodedBytes, err := base64.StdEncoding.DecodeString(transactionData.EncodedData)

	unmarshalledTransaction, unmarshalledInputData, err := getUnmarshalledTransactionAndInputData(decodedBytes)
	if err != nil {
		return fmt.Errorf("an error occured when attempting to unmarshal the decoded transaction byte array. \nerr: %w", err)
	}

	marshalledTransaction, marshalledInputData, err := marshalTransactionAndInputDataToString(unmarshalledTransaction, unmarshalledInputData)
	if err != nil {
		return fmt.Errorf("an error occurred when attempting to marshal the structs back to a json string. \nerr: %w", err)
	}

	fmt.Println("------transaction------")
	fmt.Println(marshalledTransaction)
	fmt.Println("------input data------")
	fmt.Println(marshalledInputData)

	//compare the transaction marshalled back to a string with the raw json from the json file
	result, diffString := compareJson(transactionData.Transaction, []byte(marshalledTransaction))
	//TODO- add another comparison comparing the input data from the file vs the marshalled input data from core

	if result == jsondiff.NoMatch {
		transactionDiffs += 1
		fmt.Println("diff found between json from app vs marshalled json in core, writing to file")
		// TODO- add functionality to write diff to file here as it will be tough to analyse everything in cli

		//diff here for debugging purposes for now
		fmt.Println(diffString)
	}

	return nil
}
