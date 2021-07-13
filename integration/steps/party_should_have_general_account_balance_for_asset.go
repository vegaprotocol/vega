package steps

import (
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func PartyShouldHaveGeneralAccountBalanceForAsset(
	broker *stubs.BrokerStub,
	party, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetPartyGeneralAccount(party, asset)
	if err != nil {
		return err
	}

	if acc.Balance != balance {
		return fmt.Errorf("invalid general account balance for asset(%s) for party(%s), expected(%d) got(%d)",
			asset, party, balance, acc.Balance,
		)
	}

	return nil
}
