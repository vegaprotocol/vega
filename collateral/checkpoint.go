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

var partyOverrides = map[string]types.AccountType{
	systemOwner: types.AccountTypeGlobalInsurance,
	systemOwner + types.AccountTypeGlobalReward.String():                   types.AccountTypeGlobalReward,
	systemOwner + types.AccountTypeMakerFeeReward.String():                 types.AccountTypeMakerFeeReward,
	systemOwner + types.AccountTypeTakerFeeReward.String():                 types.AccountTypeTakerFeeReward,
	systemOwner + types.AccountTypeLPFeeReward.String():                    types.AccountTypeLPFeeReward,
	systemOwner + types.AccountTypeMarketProposerReward.String():           types.AccountTypeMarketProposerReward,
	systemOwner + types.AccountTypeFeesInfrastructure.String():             types.AccountTypeFeesInfrastructure,
	systemOwner + systemOwner + types.AccountTypePendingTransfers.String(): types.AccountTypePendingTransfers,
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	msg := checkpoint.Collateral{}
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}
	for _, balance := range msg.Balances {
		ub, _ := num.UintFromString(balance.Balance, 10)
		// for backward compatibility check both - after this is already out checkpoints will always have the type for global accounts
		if tp, ok := partyOverrides[balance.Party]; ok {
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
			types.AccountTypeLPFeeReward, types.AccountTypeMakerFeeReward, types.AccountTypeTakerFeeReward,
			types.AccountTypeMarketProposerReward, types.AccountTypeFeesInfrastructure, types.AccountTypePendingTransfers:
			owner := acc.Owner
			// handle special accounts separately.
			if owner == systemOwner {
				for k, v := range partyOverrides {
					if acc.Type == v {
						owner = k
					}
				}
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
