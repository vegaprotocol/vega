package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func TraderShouldHaveOneMarginAccountPerMarket(
	broker *stubs.BrokerStub,
	owner string,
) error {
	assets := map[string]struct{}{}

	accounts := broker.GetAccounts()

	for _, acc := range accounts {
		if acc.Owner == owner && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			if _, ok := assets[acc.MarketId]; ok {
				return errMultipleMarginAccountForMarket(owner, acc)
			}
			assets[acc.MarketId] = struct{}{}
		}
	}
	return nil
}

func errMultipleMarginAccountForMarket(owner string, acc types.Account) error {
	return fmt.Errorf("trader=%v have multiple account for market=%v", owner, acc.MarketId)
}
