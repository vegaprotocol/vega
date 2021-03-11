package steps

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/vega/collateral"
)

func WithdrawFromAccount(collateral *collateral.Engine, trader, amountStr, asset string) error {
	amount, err := strconv.ParseUint(amountStr, 10, 0)
	// row.0 = traderID, row.1 = amount to topup
	if err != nil {
		return err
	}

	if _, err := collateral.LockFundsForWithdraw(
		context.Background(), trader, asset, amount,
	); err != nil {
		return err
	}

	_, err = collateral.Withdraw(
		context.Background(), trader, asset, amount,
	)
	return err
}
