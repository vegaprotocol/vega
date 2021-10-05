package steps

import (
	"fmt"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheSettlementAccountShouldHaveBalanceForMarket(
	broker *stubs.BrokerStub,
	amountStr, market string,
) error {
	amount := parseSettlementAccountBalance(amountStr)

	acc, err := broker.GetMarketSettlementAccount(market)
	if err != nil {
		return errCannotGetSettlementAccountForMarket(market, err)
	}

	if amount != stringToU64(acc.Balance) {
		return errInvalidSettlementAccountBalanceForMarket(amount, acc)
	}
	return nil
}

func parseSettlementAccountBalance(amountStr string) uint64 {
	amount, err := U64(amountStr)
	panicW("balance", err)
	return amount
}

func errCannotGetSettlementAccountForMarket(market string, err error) error {
	return fmt.Errorf("couldn't get settlement account for market(%s): %s", market, err.Error())
}

func errInvalidSettlementAccountBalanceForMarket(amount uint64, acc types.Account) error {
	return fmt.Errorf("invalid balance for market settlement account, expected %v, got %v", amount, acc.Balance)
}
