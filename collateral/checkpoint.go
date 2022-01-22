package collateral

import (
	"context"
	"sort"
	"strings"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/proto"
)

func (e *Engine) Name() types.CheckpointName {
	return types.CollateralCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	msg := &checkpoint.Collateral{
		Balances: e.getCheckpointBalances(),
	}
	ret, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	msg := checkpoint.Collateral{}
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}
	for _, balance := range msg.Balances {
		ub, _ := num.UintFromString(balance.Balance, 10)
		// for backward compatibility check both - after this is already out checkpoints will always have the type for global accounts
		if balance.Party == systemOwner || balance.Party == systemOwner+types.AccountTypeGlobalReward.String() || balance.Party == systemOwner+types.AccountTypeFeesInfrastructure.String() || balance.Party == systemOwner+types.AccountTypePendingTransfers.String() {
			tp := types.AccountTypeGlobalInsurance
			if balance.Party == systemOwner+types.AccountTypeGlobalReward.String() {
				tp = types.AccountTypeGlobalReward
			} else if balance.Party == systemOwner+types.AccountTypeFeesInfrastructure.String() {
				tp = types.AccountTypeFeesInfrastructure
			} else if balance.Party == systemOwner+types.AccountTypePendingTransfers.String() {
				tp = types.AccountTypePendingTransfers
			}

			accID := e.accountID(noMarket, systemOwner, balance.Asset, tp)
			if _, err := e.GetAccountByID(accID); err != nil {
				// this account is created when the asset is enabled. If we can't get this account,
				// then the asset is not yet enabled and we have a problem...
				return err
			}
			e.UpdateBalance(ctx, accID, ub)
			continue
		}
		accID := e.accountID(noMarket, balance.Party, balance.Asset, types.AccountTypeGeneral)
		if _, err := e.GetAccountByID(accID); err != nil {
			accID, _ = e.CreatePartyGeneralAccount(ctx, balance.Party, balance.Asset)
		}
		e.UpdateBalance(ctx, accID, ub)
	}
	return nil
}

// get all balances for checkpoint.
func (e *Engine) getCheckpointBalances() []*checkpoint.AssetBalance {
	// party -> asset -> balance
	balances := make(map[string]map[string]*num.Uint, len(e.accs))
	for _, acc := range e.accs {
		if acc.Balance.IsZero() {
			continue
		}
		switch acc.Type {
		case types.AccountTypeMargin, types.AccountTypeGeneral, types.AccountTypeBond,
			types.AccountTypeInsurance, types.AccountTypeGlobalInsurance, types.AccountTypeGlobalReward,
			types.AccountTypeFeesInfrastructure, types.AccountTypePendingTransfers:
			owner := acc.Owner
			// handle reward accounts separately.
			if owner == systemOwner && (acc.Type == types.AccountTypeGlobalReward || acc.Type == types.AccountTypeFeesInfrastructure || acc.Type == types.AccountTypePendingTransfers) {
				owner = owner + acc.Type.String()
			}

			assets, ok := balances[owner]
			if !ok {
				assets = map[string]*num.Uint{}
				balances[owner] = assets
			}
			balance, ok := assets[acc.Asset]
			if !ok {
				balance = num.Zero()
				assets[acc.Asset] = balance
			}
			balance.AddSum(acc.Balance)
		case types.AccountTypeSettlement:
			if !acc.Balance.IsZero() {
				e.log.Panic("Settlement balance is not zero",
					logging.String("market-id", acc.MarketID))
			}
		}
	}

	out := make([]*checkpoint.AssetBalance, 0, len(balances))
	for owner, assets := range balances {
		for asset, balance := range assets {
			out = append(out, &checkpoint.AssetBalance{
				Party:   owner,
				Asset:   asset,
				Balance: balance.String(),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		switch strings.Compare(out[i].Party, out[j].Party) {
		case -1:
			return true
		case 1:
			return false
		}
		return out[i].Asset < out[j].Asset
	})
	return out
}
