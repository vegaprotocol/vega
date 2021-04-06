package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheCumulatedBalanceForAllAccountsShouldBeWorth(broker *stubs.BrokerStub, rawAmount string) error {
	amount, err := U64(rawAmount)
	if err != nil {
		panicW("balance", err)
	}

	var cumulatedBalance uint64
	accounts := getAccounts(broker)
	for _, v := range accounts {
		// remove vote token
		if v.Asset != "VOTE" {
			cumulatedBalance += v.Balance
		}
	}

	if amount != cumulatedBalance {
		return fmt.Errorf("expected cumulated balance to be %v but found %v",
			amount, cumulatedBalance,
		)
	}
	return nil
}
