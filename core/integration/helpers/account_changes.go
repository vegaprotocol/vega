package helpers

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/libs/num"
	types "code.vegaprotocol.io/vega/protos/vega"
)

// ReconcileAccountChanges takes account balances before the step, modifies them based on supplied transfer, deposits, withdrawals as well as insurance pool balance changes intitiated by test code (not regular flow), and then compares them to account balances after the step.
func ReconcileAccountChanges(collateralEngine *collateral.Engine, before, after []types.Account, insurancePoolDeposits map[string]*num.Int, transfersInBetween []*types.LedgerEntry) error {
	bmp, err := mapFromAccount(before)
	if err != nil {
		return err
	}
	amp, err := mapFromAccount(after)
	if err != nil {
		return err
	}

	for acc, amt := range insurancePoolDeposits {
		if _, ok := bmp[acc]; !ok {
			bmp[acc] = num.IntZero()
		}
		bmp[acc].Add(amt)
	}

	for _, t := range transfersInBetween {
		amt, err := stringToBigInt(t.Amount)
		if err != nil {
			return err
		}
		from := t.FromAccount.ID()
		to := t.ToAccount.ID()
		if amt.IsPositive() {
			if _, ok := bmp[from]; !ok {
				bmp[from] = num.IntZero()
			}
			if _, ok := bmp[to]; !ok {
				bmp[to] = num.IntZero()
			}
			bmp[from].Sub(amt)
			bmp[to].Add(amt)
		}
	}

	keys := make([]string, 0, len(amp))
	for k := range amp {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, acc := range keys {
		value := amp[acc]
		reconciledValue := bmp[acc]
		if !value.IsZero() && (reconciledValue == nil || !reconciledValue.EQ(value)) {
			return fmt.Errorf("'%s' account balance: '%v', expected: '%v'", acc, value, reconciledValue)
		}
	}

	return checkAgainstCollateralEngineState(collateralEngine, amp)
}

func checkAgainstCollateralEngineState(collateralEngine *collateral.Engine, accounts map[string]*num.Int) error {
	for id, expectedValue := range accounts {
		acc, err := collateralEngine.GetAccountByID(id)
		if err != nil {
			if expectedValue.IsZero() {
				continue
			}
			return err
		}
		actualValue := acc.Balance
		if acc.Type == types.AccountType_ACCOUNT_TYPE_EXTERNAL {
			// we don't sent account events for external accounts
			continue
		}
		if !expectedValue.EQ(num.IntFromUint(actualValue, true)) {
			return fmt.Errorf("invalid balance for account '%s', expected: '%s', got: '%s'", id, expectedValue, actualValue)
		}
	}
	return nil
}

func mapFromAccount(accs []types.Account) (map[string]*num.Int, error) {
	mp := make(map[string]*num.Int, len(accs))
	var err error
	for _, a := range accs {
		mp[a.GetId()], err = stringToBigInt(a.Balance)
		if err != nil {
			return nil, err
		}
	}
	return mp, nil
}

func stringToBigInt(amnt string) (*num.Int, error) {
	amt, b := num.IntFromString(amnt, 10)
	if b {
		return nil, fmt.Errorf("error encountered during conversion of '%s' to num.Int", amnt)
	}
	return amt, nil
}
