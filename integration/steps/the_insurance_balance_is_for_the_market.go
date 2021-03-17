package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func TheInsurancePoolBalanceIsForTheMarket(
	broker *stubs.BrokerStub,
	rawAmount, market string,
) error {

	amount := parseExpectedInsurancePoolBalance(rawAmount)

	acc, err := broker.GetMarketInsurancePoolAccount(market)
	if err != nil {
		return errCannotGetInsurancePoolAccountForMarket(market, err)
	}

	if amount != acc.Balance {
		return errInvalidMarketInsurancePoolBalance(amount, acc)
	}
	return nil
}

func parseExpectedInsurancePoolBalance(rawAmount string) uint64 {
	amount, err := U64(rawAmount)
	panicW("balance", err)
	return amount
}

func errCannotGetInsurancePoolAccountForMarket(market string, err error) error {
	return fmt.Errorf("couldn't get insurance pool account for market(%s): %s", market, err.Error())
}

func errInvalidMarketInsurancePoolBalance(amount uint64, acc types.Account) error {
	return fmt.Errorf(
		"invalid balance for market insurance pool, expected %v, got %v",
		amount, acc.Balance,
	)
}
