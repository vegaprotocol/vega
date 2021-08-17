package collateral

import (
	"context"

	vpb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

const SnapshotName = "collateral"

func (e *Engine) Name() string {
	return SnapshotName
}

func (e *Engine) Checkpoint() []byte {
	balances := e.getPartyBalances()
	msg := &vpb.Collateral{
		Parties: make(map[string]*vpb.Balance, len(balances)),
	}
	for party, assets := range balances {
		bal := &vpb.Balance{
			Balance: make(map[string][]byte, len(assets)),
		}
		for a, b := range assets {
			bal.Balance[a] = []byte(b.String()) // either update the proto to use string, or use [32]byte reliably
		}
		msg.Parties[party] = bal
	}
	ret, err := vpb.Marshal(msg)
	if err != nil {
		e.log.Panic("Error marshalling snapshot data for collateral engine",
			logging.Error(err))
	}
	return ret
}

func (e *Engine) Load(checkpoint, _ []byte) error {
	msg := vpb.Collateral{}
	if err := vpb.Unmarshal(checkpoint, &msg); err != nil {
		return err
	}
	for party, balances := range msg.Parties {
		for a, bal := range balances.Balance {
			ub := num.UintFromBytes(bal)
			// ub, _ := num.UintFromString(bal, 10)
			accID := e.accountID(noMarket, party, a, types.AccountTypeGeneral)
			if _, err := e.GetAccountByID(accID); err != nil {
				accID, _ = e.CreatePartyGeneralAccount(context.Background(), party, a)
			}
			e.UpdateBalance(context.Background(), accID, ub)
		}
	}
	return nil
}

// get party balances per asset
func (e *Engine) getPartyBalances() map[string]map[string]*num.Uint {
	ret := make(map[string]map[string]*num.Uint, len(e.accs))
	for party, accs := range e.partiesAccs {
		balances := map[string]*num.Uint{}
		for _, acc := range accs {
			switch acc.Type {
			case types.AccountTypeMargin, types.AccountTypeGeneral, types.AccountTypeBond,
				types.AccountTypeInsurance, types.AccountTypeGlobalInsurance:
				assetBal, ok := balances[acc.Asset]
				if !ok {
					assetBal = num.Zero()
					balances[acc.Asset] = assetBal
				}
				assetBal.AddSum(acc.Balance)
			case types.AccountTypeSettlement:
				if !acc.Balance.IsZero() {
					e.log.Panic("Settlement balance is not zero",
						logging.String("market-id", acc.MarketID))
				}
			}
		}
		ret[party] = balances
	}
	return ret
}
