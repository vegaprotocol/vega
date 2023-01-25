package helpers

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
)

func ReconcileAccountChanges(before, after []types.Account, depositsInBetween []types.Deposit, transfersInBetween []*types.LedgerEntry) error {
	bmp, err := mapFromAccount(before)
	if err != nil {
		return err
	}
	amp, err := mapFromAccount(after)
	if err != nil {
		return err
	}

	for _, d := range depositsInBetween {
		genAccId := fmt.Sprintf("!%v%v%v", d.GetPartyId(), d.GetAsset(), vega.AccountType_value[vega.AccountType_ACCOUNT_TYPE_GENERAL.String()])
		if _, ok := bmp[genAccId]; !ok {
			bmp[genAccId] = num.IntZero()
		}

		amt, err := stringToBigInt(d.GetAmount())
		if err != nil {
			return err
		}
		bmp[genAccId].Add(amt)
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

	for acc, value := range amp {
		reconciledValue := bmp[acc]
		if !value.IsZero() && (reconciledValue == nil || !reconciledValue.EQ(value)) {
			return fmt.Errorf("unexpected value for '%s' account", acc)
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
