// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugTransfers(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING TRANSFERS")
	s := fmt.Sprintf("\n\t|%37s |%89s |%89s |%12s |\n", "Type", "From", "To", "Amount")
	transferEvents := broker.GetLedgerMovements(false)
	for _, e := range transferEvents {
		for _, t := range e.LedgerMovements() {
			for _, v := range t.GetEntries() {
				s += fmt.Sprintf("\t|%37s |%89s |%89s |%12s |\n", v.Type, v.FromAccount, v.ToAccount, v.Amount)
			}
		}
	}
	log.Info(s)
}
