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
	s := fmt.Sprintf("\n\t|%40s |%40s |%25s |%25s |%15s |\n", "Type", "Reference", "From", "To", "Amount")
	transferEvents := broker.GetTransferResponses()
	for _, e := range transferEvents {
		for _, t := range e.TransferResponses() {
			for _, v := range t.GetTransfers() {
				s += fmt.Sprintf("\t|%40s |%40s |%25s |%25s |%15s |\n", v.Type, v.Reference, v.FromAccount, v.ToAccount, v.Amount)
			}
		}
	}
	log.Info(s)
}
