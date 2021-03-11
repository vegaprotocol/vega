package steps

import (
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func SettlementAccountBalanceIsForMarket(broker *stubs.BrokerStub, amountStr, market string) error {
	amount, _ := strconv.ParseUint(amountStr, 10, 0)
	acc, err := broker.GetMarketSettlementAccount(market)
	if err != nil {
		return err
	}
	if amount != acc.Balance {
		return fmt.Errorf("invalid balance for market settlement account, expected %v, got %v", amount, acc.Balance)
	}
	return nil
}