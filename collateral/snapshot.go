package collateral

import (
	"context"
	"sort"

	vpb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func (e *Engine) Name() types.CheckpointName {
	return types.CollateralCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	msg := &vpb.Collateral{
		Balances: e.getSnapshotBalances(),
	}
	ret, err := vpb.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (e *Engine) Load(checkpoint []byte) error {
	msg := vpb.Collateral{}
	if err := vpb.Unmarshal(checkpoint, &msg); err != nil {
		return err
	}
	for _, balance := range msg.Balances {
		ub, _ := num.UintFromString(balance.Balance, 10)
		if balance.Party == systemOwner {
			accID := e.accountID(noMarket, systemOwner, balance.Asset, types.AccountTypeGlobalInsurance)
			if _, err := e.GetAccountByID(accID); err != nil {
				// this account is created when the asset is enabled. If we can't get this account,
				// then the asset is not yet enabled and we have a problem...
				return err
			}
			e.UpdateBalance(context.Background(), accID, ub)
			continue
		}
		accID := e.accountID(noMarket, balance.Party, balance.Asset, types.AccountTypeGeneral)
		if _, err := e.GetAccountByID(accID); err != nil {
			accID, _ = e.CreatePartyGeneralAccount(context.Background(), balance.Party, balance.Asset)
		}
		e.UpdateBalance(context.Background(), accID, ub)
	}
	return nil
}

// get all balances for snapshot
func (e *Engine) getSnapshotBalances() []*vpb.AssetBalance {
	parties := make([]string, 0, len(e.partiesAccs))
	pbal := make(map[string][]*vpb.AssetBalance, len(e.partiesAccs))
	entries := 0
	for party, accs := range e.partiesAccs {
		assets := make([]string, 0, len(accs))
		balances := map[string]*num.Uint{}
		for _, acc := range accs {
			switch acc.Type {
			case types.AccountTypeMargin, types.AccountTypeGeneral, types.AccountTypeBond,
				types.AccountTypeInsurance, types.AccountTypeGlobalInsurance:
				assetBal, ok := balances[acc.Asset]
				if !ok {
					assetBal = num.Zero()
					balances[acc.Asset] = assetBal
					assets = append(assets, acc.Asset)
				}
				assetBal.AddSum(acc.Balance)
			case types.AccountTypeSettlement:
				if !acc.Balance.IsZero() {
					e.log.Panic("Settlement balance is not zero",
						logging.String("market-id", acc.MarketID))
				}
			}
		}
		ln := len(assets)
		if ln == 0 {
			continue
		}
		entries += ln
		pbal[party] = make([]*vpb.AssetBalance, 0, len(assets))
		parties = append(parties, party)
		// sort by asset -> each party will have their balances appended in alphabetic order
		sort.Strings(assets)
		for _, a := range assets {
			bal := balances[a]
			pbal[party] = append(pbal[party], &vpb.AssetBalance{
				Party:   party,
				Asset:   a,
				Balance: bal.String(),
			})
		}
	}
	ret := make([]*vpb.AssetBalance, 0, entries)
	sort.Strings(parties)
	for _, party := range parties {
		ret = append(ret, pbal[party]...)
	}
	return ret
}
