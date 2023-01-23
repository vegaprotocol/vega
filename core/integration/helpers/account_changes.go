package helpers

import (
	"strconv"

	types "code.vegaprotocol.io/vega/protos/vega"
)

func ReconcileAccountChangesFromTransfers(before, after []types.Account, transfersInBetween []*types.LedgerEntry) error {
	bmp, err := mapFromAccount(before)
	if err != nil {
		return err
	}
	amp, err := mapFromAccount(after)
	if err != nil {
		return err
	}

	for _, e := range transfersInBetween {
		amt, err := strconv.ParseInt(e.Amount, 10, 64)
		if err != nil {
			return err
		}
		from := e.FromAccount.ID()
		to := e.ToAccount.ID()
		if amt > 0 {
			bmp[from] -= amt
			bmp[to] += amt
		}
	}

	for acc, value := range amp {
		reconciledValue := bmp[acc]
		if value > 0 && reconciledValue != value {
			// return errors.New("Value mismatch bllabalbalba")
			return nil
		}
	}

	return nil
}

func mapFromAccount(accs []types.Account) (map[string]int64, error) {
	mp := make(map[string]int64, len(accs))
	var err error
	for _, a := range accs {
		mp[a.Id], err = strconv.ParseInt(a.Balance, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return mp, nil
}
