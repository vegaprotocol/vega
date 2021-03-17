package steps

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/vega/collateral"
)

func DepositIntoAccount(collateral *collateral.Engine, owner, amountstr, asset string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	_, err := collateral.Deposit(
		context.Background(), owner, asset, amount,
	)
	return err
}
