package inspecttx_helpers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/nsf/jsondiff"
)

type Result struct {
	Match               bool
	TransactionJson     ComparableJson
	InputDataJson       ComparableJson
	TransactionHtmlDiff string
	InputDataHtmlDiff   string
}

func TransactionMatch(transactionData TransactionData) (Result, error) {
	comparisonData := Result{}
	decodedBytes, err := base64.StdEncoding.DecodeString(transactionData.EncodedData)
	if err != nil {
		return comparisonData, fmt.Errorf("error occurred when decoding the transaction encoded data.\nerr: %v", err)
	}

	unmarshalledTransaction, unmarshalledInputData, err := unmarshalTransaction(decodedBytes)
	if err != nil {
		return comparisonData, fmt.Errorf("an error occurred when attempting to unmarshal the decoded transaction byte array. \nerr: %v", err)
	}

	transactionWrapper := &TransactionAlias{Transaction: unmarshalledTransaction}
	marshalledTransaction, marshalledInputData, err := marshalTransactionAndInputDataToString(transactionWrapper, unmarshalledInputData)
	if err != nil {
		return comparisonData, fmt.Errorf("an error occurred when attempting to marshal the structs back to a json string. \nerr: %v", err)
	}

	transactionCompareResult, transactionHtmlDiff := compareJson(transactionData.Transaction, []byte(marshalledTransaction))
	inputDataCompareResult, inputDataHtmlDiff := compareJson(transactionData.InputData, []byte(marshalledInputData))

	if transactionCompareResult == jsondiff.NoMatch || inputDataCompareResult == jsondiff.NoMatch {
		comparisonData.Match = false
		if transactionCompareResult == jsondiff.NoMatch {
			comparableTransactionJson := ComparableJson{
				OriginalJson: transactionData.Transaction,
				CoreJson:     json.RawMessage(marshalledTransaction),
				DiffType:     Transaction,
			}

			logrus.Errorf("transaction data did not match")
			comparisonData.TransactionJson = comparableTransactionJson
			comparisonData.TransactionHtmlDiff = transactionHtmlDiff
		}

		if inputDataCompareResult == jsondiff.NoMatch {
			comparableInputDataJson := ComparableJson{
				OriginalJson: transactionData.InputData,
				CoreJson:     json.RawMessage(marshalledInputData),
				DiffType:     InputData,
			}

			logrus.Errorf("input data did not match")
			comparisonData.InputDataJson = comparableInputDataJson
			comparisonData.InputDataHtmlDiff = inputDataHtmlDiff
		}
		return comparisonData, nil
	}

	comparisonData.Match = true
	return comparisonData, nil
}
