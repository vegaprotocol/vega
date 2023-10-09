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

func DebugAccounts(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING ACCOUNTS")
	s := fmt.Sprintf("\n\t|%10s |%15s |%15s |%10s |%25s |\n", "MarketId", "Owner", "Balance", "Asset", "AccountId")
	accounts := broker.GetAccounts()
	for _, a := range accounts {
		s += fmt.Sprintf("\t|%10s |%15s |%15s |%10s |%25s |\n", a.MarketId, a.Owner, a.Balance, a.Asset, a.Id)
	}
	log.Info(s)
}
