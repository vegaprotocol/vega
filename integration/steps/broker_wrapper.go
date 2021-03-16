package steps

import (
	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func getAccounts(broker *stubs.BrokerStub) []types.Account {
	events := broker.GetAccounts()
	accounts := make([]types.Account, 0, len(events))
	for _, a := range events {
		accounts = append(accounts, a.Account())
	}
	return accounts
}
