// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

const (
	separator            = "___"
	vestingAccountPrefix = "vesting"
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

var partyOverrideAlias = map[string]string{
	systemOwner + types.AccountTypeNetworkTreasury.String(): systemOwner,
}

var partyOverrides = map[string]types.AccountType{
	systemOwner: types.AccountTypeNetworkTreasury,
	systemOwner + types.AccountTypeGlobalInsurance.String():        types.AccountTypeGlobalInsurance,
	systemOwner + types.AccountTypeGlobalReward.String():           types.AccountTypeGlobalReward,
	systemOwner + types.AccountTypeMakerReceivedFeeReward.String(): types.AccountTypeMakerReceivedFeeReward,
	systemOwner + types.AccountTypeMakerPaidFeeReward.String():     types.AccountTypeMakerPaidFeeReward,
	systemOwner + types.AccountTypeLPFeeReward.String():            types.AccountTypeLPFeeReward,
	systemOwner + types.AccountTypeAverageNotionalReward.String():  types.AccountTypeAverageNotionalReward,
	systemOwner + types.AccountTypeRelativeReturnReward.String():   types.AccountTypeRelativeReturnReward,
	systemOwner + types.AccountTypeReturnVolatilityReward.String(): types.AccountTypeReturnVolatilityReward,
	systemOwner + types.AccountTypeValidatorRankingReward.String(): types.AccountTypeValidatorRankingReward,
	systemOwner + types.AccountTypeMarketProposerReward.String():   types.AccountTypeMarketProposerReward,
	systemOwner + types.AccountTypeFeesInfrastructure.String():     types.AccountTypeFeesInfrastructure,
	systemOwner + types.AccountTypePendingTransfers.String():       types.AccountTypePendingTransfers,
	systemOwner + types.AccountTypeRealisedReturnReward.String():   types.AccountTypeRealisedReturnReward,
	systemOwner + types.AccountTypeEligibleEntitiesReward.String(): types.AccountTypeEligibleEntitiesReward,
}

var tradingRewardAccountTypes = map[types.AccountType]struct{}{
	types.AccountTypeMakerReceivedFeeReward: {},
	types.AccountTypeMakerPaidFeeReward:     {},
	types.AccountTypeLPFeeReward:            {},
	types.AccountTypeMarketProposerReward:   {},
	types.AccountTypeAverageNotionalReward:  {},
	types.AccountTypeRelativeReturnReward:   {},
	types.AccountTypeReturnVolatilityReward: {},
	types.AccountTypeValidatorRankingReward: {},
	types.AccountTypeRealisedReturnReward:   {},
	types.AccountTypeEligibleEntitiesReward: {},
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	msg := checkpoint.Collateral{}
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}

	ledgerMovements := []*types.LedgerMovement{}
	assets := map[string]struct{}{}

	for _, balance := range msg.Balances {
		ub, _ := num.UintFromString(balance.Balance, 10)
		isVesting := strings.HasPrefix(balance.Party, vestingAccountPrefix)
		if isVesting {
			balance.Party = strings.TrimPrefix(balance.Party, vestingAccountPrefix)
		}
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
			assets[balance.Asset] = struct{}{}
			ledgerMovements = append(ledgerMovements, lm)
			continue
		}
		var (
			lm  *types.LedgerMovement
			err error
		)

		if isVesting {
			accID := e.accountID(market, balance.Party, balance.Asset, types.AccountTypeVestingRewards)
			if _, err := e.GetAccountByID(accID); err != nil {
				_ = e.GetOrCreatePartyVestingRewardAccount(ctx, balance.Party, balance.Asset)
			}
			lm, err = e.RestoreCheckpointBalance(
				ctx, noMarket, balance.Party, balance.Asset, types.AccountTypeVestingRewards, ub.Clone())
			if err != nil {
				return err
			}

			e.addToVesting(balance.Party, balance.Asset, ub.Clone())
		} else {
			accID := e.accountID(market, balance.Party, balance.Asset, types.AccountTypeGeneral)
			if _, err := e.GetAccountByID(accID); err != nil {
				_, _ = e.CreatePartyGeneralAccount(ctx, balance.Party, balance.Asset)
			}
			lm, err = e.RestoreCheckpointBalance(
				ctx, noMarket, balance.Party, balance.Asset, types.AccountTypeGeneral, ub.Clone())
			if err != nil {
				return err
			}
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
		// vesting rewards needs to be stored separately
		// so that vesting can be started again
		case types.AccountTypeVestingRewards:
			owner := vestingAccountPrefix + acc.Owner

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

		case types.AccountTypeMargin, types.AccountTypeOrderMargin, types.AccountTypeGeneral, types.AccountTypeHolding, types.AccountTypeBond, types.AccountTypeFeesLiquidity,
			types.AccountTypeInsurance, types.AccountTypeGlobalReward, types.AccountTypeLiquidityFeesBonusDistribution, types.AccountTypeLPLiquidityFees,
			types.AccountTypeLPFeeReward, types.AccountTypeMakerReceivedFeeReward, types.AccountTypeMakerPaidFeeReward,
			types.AccountTypeMarketProposerReward, types.AccountTypeFeesInfrastructure, types.AccountTypePendingTransfers,
			types.AccountTypeNetworkTreasury, types.AccountTypeGlobalInsurance, types.AccountTypeVestedRewards,
			types.AccountTypeAverageNotionalReward, types.AccountTypeRelativeReturnReward, types.AccountTypeRealisedReturnReward,
			types.AccountTypeReturnVolatilityReward, types.AccountTypeValidatorRankingReward, types.AccountTypeEligibleEntitiesReward:
			owner := acc.Owner
			// NB: market insurance accounts funds will flow implicitly using this logic into the network treasury for the asset
			// similarly LP Fee bonus distribution bonus account would fall over into the network treasury of the asset.
			if owner == systemOwner {
				for k, v := range partyOverrides {
					if acc.Type == v {
						owner = k
					}
				}
				if acc.Type == types.AccountTypeInsurance {
					// let the market insurnace fall into the global insurance account
					owner = systemOwner + types.AccountTypeGlobalInsurance.String()
				}
			}

			// NB: for market based reward accounts we don't want to move the funds to the network treasury but rather keep them
			if acc.Type == types.AccountTypeLPFeeReward ||
				acc.Type == types.AccountTypeMakerReceivedFeeReward ||
				acc.Type == types.AccountTypeMakerPaidFeeReward ||
				acc.Type == types.AccountTypeMarketProposerReward ||
				acc.Type == types.AccountTypeAverageNotionalReward ||
				acc.Type == types.AccountTypeRelativeReturnReward ||
				acc.Type == types.AccountTypeReturnVolatilityReward ||
				acc.Type == types.AccountTypeValidatorRankingReward ||
				acc.Type == types.AccountTypeRealisedReturnReward ||
				acc.Type == types.AccountTypeEligibleEntitiesReward {
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
