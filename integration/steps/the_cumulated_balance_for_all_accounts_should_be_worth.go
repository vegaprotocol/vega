package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/types/num"
)

func TheCumulatedBalanceForAllAccountsShouldBeWorth(broker *stubs.BrokerStub, rawAmount string) error {
	amount, _ := num.UintFromString(rawAmount, 10)

	cumulatedBalance := num.Zero()
	accounts := broker.GetAccounts()
	for _, v := range accounts {
		// remove vote token
		if v.Asset != "VOTE" {
			b, _ := num.UintFromString(v.Balance, 10)
			cumulatedBalance.AddSum(b)
		}
	}

	if !amount.EQ(cumulatedBalance) {
		return fmt.Errorf("expected cumulated balance to be %v but found %v",
			amount, cumulatedBalance,
		)
	}
	return nil
}
