package steps

import (
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func PartyShouldHaveGeneralAccountBalanceForAsset(
	broker *stubs.BrokerStub,
	trader, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetPartyGeneralAccount(trader, asset)
	if err != nil {
		return err
	}

	if acc.Balance != balance {
		return fmt.Errorf("invalid general account balance for asset(%s) for trader(%s), expected(%d) got(%d)",
			asset, trader, balance, acc.Balance,
		)
	}

	return nil
}
