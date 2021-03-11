package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func HaveOnlyOneMarginAccountPerMarket(broker *stubs.BrokerStub, owner string) error {
	assets := map[string]struct{}{}

	accs := broker.GetAccounts()
	data := make([]types.Account, 0, len(accs))
	for _, a := range accs {
		data = append(data, a.Account())
	}
	for _, acc := range data {
		if acc.Owner == owner && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			if _, ok := assets[acc.MarketId]; ok {
				return fmt.Errorf("trader=%v have multiple account for market=%v", owner, acc.MarketId)
			}
			assets[acc.MarketId] = struct{}{}
		}
	}
	return nil
}
