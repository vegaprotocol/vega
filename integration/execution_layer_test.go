package core_test

import (
	"fmt"
	"strconv"
)

func theInsurancePoolInitialBalanceForTheMarketsIs(amountstr string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	execsetup = getExecutionSetupEmptyWithInsurancePoolBalance(amount)
	return nil
}

func generalAccountForAssetBalanceIs(trader, asset, balancestr string) error {
	balance, _ := strconv.ParseUint(balancestr, 10, 0)
	acc, err := execsetup.broker.GetTraderGeneralAccount(trader, asset)
	if err != nil {
		return err
	}

	if acc.Balance != balance {
		return fmt.Errorf("invalid general account balance for asset(%s) for trader(%s), expected(%d) got(%d)",
			asset, trader, balance, acc.Balance,
		)
	}

	return nil
}

