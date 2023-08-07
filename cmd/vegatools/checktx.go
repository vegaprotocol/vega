package tools

import (
	"fmt"

	inspecttx2 "code.vegaprotocol.io/vega/vegatools/inspecttx"

	"github.com/sirupsen/logrus"

	"code.vegaprotocol.io/vega/core/config"
)

var (
	transactionDiffs     int
	transactionsAnalysed int
	transactionsPassed   int
)

type checkTxCmd struct {
	config.OutputFlag
	Diffs        string `description:"The output directory for detailed reporting on diffs"             long:"diffs"    short:"d"`
	Transactions string `description:"Path to the transaction json files"                               long:"transactions"    short:"t" required:"true"`
}

func (opts *checkTxCmd) Execute(_ []string) error {
	transactionsAnalysed = 0
	transactionsPassed = 0
	transactionDiffs = 0

	transactionFiles, err := inspecttx2.GetFilesInDirectory(opts.Transactions)
	if err != nil {
		return fmt.Errorf("error when attempting to get files in the given directory. \nerr: %v", err)
	}

	for _, file := range transactionFiles {
		transactionData, err := inspecttx2.GetTransactionDataFromFile(file)
		if err != nil {
			return fmt.Errorf("error reading transaction file '%s'\nerr: %v", file, err)
		}

		logrus.Infof("inspecting transaction in '%s'", file)
		result, err := inspecttx2.TransactionMatch(transactionData)
		if err != nil {
			return fmt.Errorf("error when attempting to inspect transaction in file '%s' \nerr: %v", file, err)
		}

		if result.Match {
			transactionsPassed += 1
		} else {
			transactionDiffs += 1
			inspecttx2.WriteDiffsToFile(file, opts.Diffs, result)
		}
		transactionsAnalysed += 1
	}

	logrus.Infof("transactions analysed: %d, transactions passed: %d, transactions failed: %d", transactionsAnalysed, transactionsPassed, transactionDiffs)
	if transactionDiffs != 0 {
		return fmt.Errorf("there were diffs in the transactions sent from your application vs the marshalled equivalents from core, check your protos are up to date. Diffs can be found in '%s'\nnumber of transactions with diffs: %d", opts.Diffs, transactionDiffs)
	}

	return nil
}
