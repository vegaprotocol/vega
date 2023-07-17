// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package collateral

import (
	"context"
	"sort"
	"strings"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
)

const separator = "___"

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

var partyOverrideAlias = map[string]string{
	systemOwner + types.AccountTypeGlobalReward.String(): systemOwner,
}

var partyOverrides = map[string]types.AccountType{
	systemOwner: types.AccountTypeGlobalReward,
	systemOwner + types.AccountTypeMakerReceivedFeeReward.String(): types.AccountTypeMakerReceivedFeeReward,
	systemOwner + types.AccountTypeMakerPaidFeeReward.String():     types.AccountTypeMakerPaidFeeReward,
	systemOwner + types.AccountTypeLPFeeReward.String():            types.AccountTypeLPFeeReward,
	systemOwner + types.AccountTypeMarketProposerReward.String():   types.AccountTypeMarketProposerReward,
	systemOwner + types.AccountTypeFeesInfrastructure.String():     types.AccountTypeFeesInfrastructure,
	systemOwner + types.AccountTypePendingTransfers.String():       types.AccountTypePendingTransfers,
}

var tradingRewardAccountTypes = map[types.AccountType]struct{}{
	types.AccountTypeMakerReceivedFeeReward: {},
	types.AccountTypeMakerPaidFeeReward:     {},
	types.AccountTypeLPFeeReward:            {},
	types.AccountTypeMarketProposerReward:   {},
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	msg := checkpoint.Collateral{}
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}

	ledgerMovements := []*types.LedgerMovement{}

	for _, balance := range msg.Balances {
		ub, _ := num.UintFromString(balance.Balance, 10)
		partyComponents := strings.Split(balance.Party, separator)
		owner := partyComponents[0]
		market := noMarket
		if len(partyComponents) > 1 {
			market = partyComponents[1]
		}

		if alias, aliasExists := partyOverrideAlias[owner]; aliasExists {
			owner = alias
		}

		// for backward compatibility check both - after this is already out checkpoints will always have the type for global accounts
		if tp, ok := partyOverrides[owner]; ok {
			accID := e.accountID(market, systemOwner, balance.Asset, tp)
			if _, ok := tradingRewardAccountTypes[tp]; ok {
				e.GetOrCreateRewardAccount(ctx, balance.Asset, market, tp)
			}
			_, err := e.GetAccountByID(accID)
			if err != nil {
				return err
			}
			lm, err := e.RestoreCheckpointBalance(
				ctx, market, systemOwner, balance.Asset, tp, ub.Clone())
			if err != nil {
				return err
			}
			ledgerMovements = append(ledgerMovements, lm)
			continue
		}
		accID := e.accountID(market, balance.Party, balance.Asset, types.AccountTypeGeneral)
		if _, err := e.GetAccountByID(accID); err != nil {
			_, _ = e.CreatePartyGeneralAccount(ctx, balance.Party, balance.Asset)
		}
		lm, err := e.RestoreCheckpointBalance(
			ctx, noMarket, balance.Party, balance.Asset, types.AccountTypeGeneral, ub.Clone())
		if err != nil {
			return err
		}
		ledgerMovements = append(ledgerMovements, lm)
	}

	if len(ledgerMovements) > 0 {
		e.broker.Send(events.NewLedgerMovements(ctx, ledgerMovements))
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
		case types.AccountTypeMargin, types.AccountTypeGeneral, types.AccountTypeHolding, types.AccountTypeBond, types.AccountTypeFeesLiquidity,
			types.AccountTypeInsurance, types.AccountTypeGlobalReward, types.AccountTypeLiquidityFeesBonusDistribution, types.AccountTypeLPLiquidityFees,
			types.AccountTypeLPFeeReward, types.AccountTypeMakerReceivedFeeReward, types.AccountTypeMakerPaidFeeReward,
			types.AccountTypeMarketProposerReward, types.AccountTypeFeesInfrastructure, types.AccountTypePendingTransfers:
			owner := acc.Owner
			// NB: market insurance accounts funds will flow implicitly using this logic into the network treasury for the asset
			// similarly LP Fee bonus distribution bonus account would fall over into the network treasury of the asset.
			if owner == systemOwner {
				for k, v := range partyOverrides {
					if acc.Type == v {
						owner = k
					}
				}
			}
			// NB: for market based reward accounts we don't want to move the funds to the network treasury but rather keep them
			if acc.Type == types.AccountTypeLPFeeReward || acc.Type == types.AccountTypeMakerReceivedFeeReward || acc.Type == types.AccountTypeMakerPaidFeeReward || acc.Type == types.AccountTypeMarketProposerReward {
				owner += separator + acc.MarketID
			}

			assets, ok := balances[owner]
			if !ok {
				assets = map[string]*num.Uint{}
				balances[owner] = assets
			}
			balance, ok := assets[acc.Asset]
			if !ok {
				balance = num.UintZero()
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
