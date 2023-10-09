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

package tools

import (
	"fmt"

	"code.vegaprotocol.io/vega/vegatools/checktx"

	"github.com/sirupsen/logrus"

	"code.vegaprotocol.io/vega/core/config"
)

type checkTxCmd struct {
	config.OutputFlag
	EncodedTransaction string `description:"The encoded transaction string to compare with vega's own encoding"                         long:"tx"    short:"t"`
	TransactionDir     string `description:"directory containing files with encoded transaction data. One encoded transaction per file" long:"txdir" short:"d"`
}

func (opts *checkTxCmd) Execute(_ []string) error {
	if opts.EncodedTransaction != "" {
		return checktx.CheckTransaction(opts.EncodedTransaction)
	}

	if opts.TransactionDir != "" {
		result, err := checktx.CheckTransactionsInDirectory(opts.TransactionDir)
		if err != nil {
			return fmt.Errorf("there was an issue when checking transactions\nerr: %w", err)
		}

		logrus.Infof("transactions analysed %d, transactions passed: %d, transactions failed: %d", result.TransactionsAnalysed, result.TransactionsPassed, result.TransactionsFailed)
		if result.TransactionsFailed > 0 {
			return fmt.Errorf("one or more transactions failed comparison")
		} else {
			return nil
		}
	}

	return fmt.Errorf("no valid arg provided")
}
