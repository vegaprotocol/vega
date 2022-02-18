package steps

import (
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func RewardAccountBalanceForAssetShouldMatch(
	broker *stubs.BrokerStub,
	accountType, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetRewardAccountBalance(accountType, asset)
	if err != nil {
		if balance == 0 {
			return nil
		}
		return err
	}

	if stringToU64(acc.Balance) != balance {
		return fmt.Errorf("invalid reward account balance for asset(%s) for account type(%s), expected(%d) got(%s)",
			asset, accountType, balance, acc.Balance,
		)
	}

	return nil
}
