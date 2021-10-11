package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugAccounts(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING ACCOUNTS")
	fmt.Printf("\t|%10s |%15s |%15s |%10s |%25s |\n", "MarketId", "Owner", "Balance", "Asset", "AccountId")
	accounts := broker.GetAccounts()
	for _, a := range accounts {
		fmt.Printf("\t|%10s |%15s |%15s |%10s |%25s |\n", a.MarketId, a.Owner, a.Balance, a.Asset, a.Id)
	}
}
