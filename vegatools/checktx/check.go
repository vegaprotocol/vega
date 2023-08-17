package checktx

import (
	"encoding/base64"
	"fmt"

	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
)

func CheckTransaction(encodedTransaction string) error {
	logrus.Infof("checking transaction...")
	unmarshalledTransaction, err := decodeAndUnmarshalTransaction(encodedTransaction)
	if err != nil {
		return fmt.Errorf("error occurred when attempting to unmarshal the encoded transaction data.\nerr: %w", err)
	}

	logrus.Infof("reencoding the transaction...")
	reEncodedTransaction, err := marshalAndEncodeTransaction(unmarshalledTransaction)
	if err != nil {
		return fmt.Errorf("error occurred when attempting to marshal and re-encode the transaction")
	}

	if !cmp.Equal(encodedTransaction, reEncodedTransaction) {
		logrus.Errorf("transactions not equal!")
		return fmt.Errorf("the reencoded transaction was not equal. \nOriginal: %s\nVegaEncoded: %s", encodedTransaction, reEncodedTransaction)
	}

	logrus.Infof("transactions equal!")
	return nil
}

type ResultData struct {
	TransactionsPassed   int
	TransactionsFailed   int
	TransactionsAnalysed int
}

func CheckTransactionsInDirectory(transactionDir string) (ResultData, error) {
	resultData := ResultData{}
	files, err := GetFilesInDirectory(transactionDir)
	if err != nil {
		return resultData, fmt.Errorf("error occurred when attempting to retrieve files from directory %s\nerr: %w", transactionDir, err)
	}

	for _, file := range files {
		transaction, err := GetEncodedTransactionFromFile(file)
		if err != nil {
			return resultData, fmt.Errorf("error getting encoded data from file '%s', the transaction could not be analysed, check your data is base64 encoded\nerr: %w", file, err)
		}

		logrus.Infof("inspecting encoded data in file %s", file)
		err = CheckTransaction(transaction)
		if err != nil {
			resultData.TransactionsFailed += 1
			logrus.Errorf("Transaction Failed! | Test file: %s | Error: %v", file, err)
		} else {
			resultData.TransactionsPassed += 1
		}
		resultData.TransactionsAnalysed += 1
	}

	return resultData, nil
}

func decodeAndUnmarshalTransaction(encodedTransaction string) (*commandspb.Transaction, error) {
	unmarshalledTransaction := &commandspb.Transaction{}
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedTransaction)
	if err != nil {
		return unmarshalledTransaction, fmt.Errorf("error occurred when decoding the encoded data.\nerr: %w", err)
	}

	if err := proto.Unmarshal(decodedBytes, unmarshalledTransaction); err != nil {
		return unmarshalledTransaction, fmt.Errorf("unable to unmarshal the decoded bytes to a transaction: %w", err)
	}

	logrus.Infof("successfully decoded and unmarshalled the transaction")

	return unmarshalledTransaction, nil
}

func marshalAndEncodeTransaction(transaction *commandspb.Transaction) (string, error) {
	pb, err := proto.Marshal(transaction)
	if err != nil {
		return "", fmt.Errorf("error when attempting to marshal Transaction struct back to a proto.\nerr: %w", err)
	}

	reEncodedTransaction := base64.StdEncoding.EncodeToString(pb)
	logrus.Infof("successfully marshalled to proto and encoded")
	return reEncodedTransaction, nil
}
