package tools

import (
	"fmt"

	inspecttx "code.vegaprotocol.io/vega/vegatools/inspecttx/inspecttx-helpers"

	"github.com/sirupsen/logrus"

	"code.vegaprotocol.io/vega/core/config"
)

var (
	transactionDiffs     int
	transactionsAnalysed int
	transactionsPassed   int
	currentFile          string
)

type checkTxCmd struct {
	config.OutputFlag
	txDirectory   string `description:"path to transaction data"                           long:"txdir"           short:"d"  required:"true"`
	diffOutputDir string `description:"path to output any diffs between transaction data" long:"diff-output-dir" short:"o" default:"./transaction-diffs"`
}

func (opts *checkTxCmd) Execute(_ []string) error {
	transactionsAnalysed = 0
	transactionsPassed = 0
	transactionDiffs = 0

	transactionFiles, err := inspecttx.GetFilesInDirectory(opts.txDirectory)
	if err != nil {
		return fmt.Errorf("error when attempting to get files in the given directory. \nerr: %v", err)
	}

	for _, file := range transactionFiles {
		currentFile = file
		transactionData, err := inspecttx.GetTransactionDataFromFile(file)
		if err != nil {
			return fmt.Errorf("error reading transaction file '%s'\nerr: %v", file, err)
		}

		logrus.Infof("inspecting transactions in '%s'", file)
		result, err := inspecttx.TransactionMatch(transactionData)
		if err != nil {
			return fmt.Errorf("error when attempting to inspect transaction in file '%s' \nerr: %v", file, err)
		}

		if result.Match {
			transactionsPassed += 1
		} else {
			transactionDiffs += 1
			inspecttx.WriteDiffsToFile(file, opts.diffOutputDir, result)

		}
		transactionsAnalysed += 1
	}

	logrus.Infof("transactions analysed: %d, transactions passed: %d, transactions failed: %d", transactionsAnalysed, transactionsPassed, transactionDiffs)
	if transactionDiffs != 0 {
		return fmt.Errorf("there were diffs in the transactions sent from your application vs the marshalled equivalents from core, check your protos are up to date. Diffs can be found in '%s'\nnumber of transactions with diffs: %d", opts.diffOutputDir, transactionDiffs)
	}

	return nil
}
