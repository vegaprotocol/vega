// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/integration/stubs"
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
