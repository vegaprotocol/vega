package collateral

import (
	"context"
	"encoding/json"

	"code.vegaprotocol.io/vega/logging"
	vpb "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"google.golang.org/protobuf/proto"
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
			Balance: make(map[string]string, len(assets)),
		}
		for a, b := range assets {
			bal.Balance[a] = b.String()
		}
		msg.Parties[party] = bal
	}
	ret, err := proto.Marshal(msg)
	if err != nil {
		e.log.Panic("Error marshalling snapshot data for collateral engine",
			logging.Error(err))
	}
	return ret
}

func (e *Engine) Checkpoint2() []byte {
	balances := e.getPartyBalances()
	numBytes := make(map[string]map[string][32]byte, len(balances))
	for p, assets := range balances {
		aBal := make(map[string][32]byte, len(assets))
		for a, b := range assets {
			aBal[a] = b.Bytes()
		}
		numBytes[p] = aBal
	}
	ret, err := json.Marshal(numBytes)
	if err != nil {
		e.log.Panic("Error marshalling snapshot data for collateral engine",
			logging.Error(err))
	}
	return ret
}

func (e *Engine) Load(checkpoint, _ []byte) error {
	msg := vpb.Collateral{}
	if err := proto.Unmarshal(checkpoint, &msg); err != nil {
		return err
	}
	for party, balances := range msg.Parties {
		for a, bal := range balances.Balance {
			ub, _ := num.UintFromString(bal, 10)
			accID := e.accountID(noMarket, party, a, types.AccountTypeGeneral)
			if _, err := e.GetAccountByID(accID); err != nil {
				accID, _ = e.CreatePartyGeneralAccount(context.Background(), party, a)
			}
			e.UpdateBalance(context.Background(), accID, ub)
		}
	}
	return inl
}

func (e *Engine) Load2(checkpoint, _ []byte) error {
	data := map[string]map[string][32]byte{}
	if err := json.Unmarshal(checkpoint, &data); err != nil {
		return err
	}
	for party, assets := range data {
		for a, bal := range assets {
			ub := num.UintFromBytes(bal[:])
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
