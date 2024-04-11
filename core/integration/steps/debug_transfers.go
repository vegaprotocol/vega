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

package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugTransfers(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING TRANSFERS")
	s := fmt.Sprintf("\n\t|%38s |%85s |%85s |%19s |\n", "Type", "From", "To", "Amount")
	transferEvents := broker.GetLedgerMovements(false)
	for _, e := range transferEvents {
		for _, t := range e.LedgerMovements() {
			for _, v := range t.GetEntries() {
				s += fmt.Sprintf("\t|%38s |%85s |%85s |%19s |\n", v.Type, v.FromAccount, v.ToAccount, v.Amount)
			}
		}
	}
	log.Info(s)
}
