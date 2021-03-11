package steps

import (
	"fmt"
	"strconv"

	types "code.vegaprotocol.io/vega/proto"
)

func TheInsurancePoolBalanceIsForTheMarket(
	broker interface {
		GetMarketInsurancePoolAccount(string) (types.Account, error)
	},
	amountstr, market string,
) error {

	amount, err := strconv.ParseUint(amountstr, 10, 0)
	panicW(err)
	acc, err := broker.GetMarketInsurancePoolAccount(market)
	if err != nil {
		return err
	}
	if amount != acc.Balance {
		return fmt.Errorf(
			"invalid balance for market insurance pool, expected %v, got %v",
			amount, acc.Balance,
		)
	}
	return nil
}
