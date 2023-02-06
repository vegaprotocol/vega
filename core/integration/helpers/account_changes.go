package helpers

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
)

// ReconcileAccountChanges takes account balances before the step, modifies them based on supplied transfer, deposits, withdrawals as well as insurance pool balance changes intitiated by test code (not regular flow), and then compares them to account balances after the step.
func ReconcileAccountChanges(before, after []types.Account, deposits []types.Deposit, withdrawals []types.Withdrawal, insurancePoolDeposits map[string]*num.Int, transfersInBetween []*types.LedgerEntry) error {
	bmp, err := mapFromAccount(before)
	if err != nil {
		return err
	}
	amp, err := mapFromAccount(after)
	if err != nil {
		return err
	}

	for _, d := range deposits {
		amt, err := stringToBigInt(d.GetAmount())
		if err != nil {
			return err
		}

		genAccId := genAccId(d.GetPartyId(), d.GetAsset())
		if _, ok := bmp[genAccId]; !ok {
			bmp[genAccId] = num.IntZero()
		}

		bmp[genAccId].Add(amt)
	}

	for _, w := range withdrawals {
		amt, err := stringToBigInt(w.GetAmount())
		if err != nil {
			return err
		}

		genAccId := genAccId(w.GetPartyId(), w.GetAsset())
		accBal, ok := bmp[genAccId]

		if !ok {
			return fmt.Errorf("account %s not found", genAccId)
		}

		if accBal.LT(amt) {
			return fmt.Errorf("account %s balance couldn't support the withdrawal specified", genAccId)
		}

		bmp[genAccId] = accBal.Sub(amt)
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

func genAccId(partyId, asset string) string {
	return fmt.Sprintf("!%v%v%v", partyId, asset, vega.AccountType_value[vega.AccountType_ACCOUNT_TYPE_GENERAL.String()])
}
