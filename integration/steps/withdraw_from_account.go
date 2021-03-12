package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/collateral"
)

func WithdrawFromAccount(
	collateral *collateral.Engine,
	trader, amountStr, asset string,
) error {
	amount := parseWithdrawAmount(amountStr)

	_, err := collateral.LockFundsForWithdraw(context.Background(), trader, asset, amount)
	if err != nil {
		return errCannotLockFundsForWithdrawal(trader, asset, amount, err)
	}

	_, err = collateral.Withdraw(context.Background(), trader, asset, amount)
	return err
}

func errCannotLockFundsForWithdrawal(trader, asset string, amount uint64, err error) error {
	return fmt.Errorf("couldn't lock funds for withdrawal of amount(%d) for trader(%s), asset(%s): %s",
		amount, trader, asset, err.Error(),
	)
}

func parseWithdrawAmount(amountStr string) uint64 {
	amount, err := U64(amountStr)
	panicW("amount", err)
	return amount
}
