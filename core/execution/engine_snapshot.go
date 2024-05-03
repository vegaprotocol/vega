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

package execution

import (
	"context"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/future"
	"code.vegaprotocol.io/vega/core/execution/spot"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
)

var marketsKey = (&types.PayloadExecutionMarkets{}).Key()

func (e *Engine) marketsStates() ([]*types.ExecMarket, []types.StateProvider) {
	mkts := len(e.futureMarketsCpy)
	if mkts == 0 {
		return nil, nil
	}
	mks := make([]*types.ExecMarket, 0, mkts)
	if prev := len(e.generatedProviders); prev < mkts {
		mkts -= prev
	}
	e.newGeneratedProviders = make([]types.StateProvider, 0, mkts*5)
	for _, m := range e.futureMarketsCpy {
		// ensure the next MTM timestamp is set correctly:
		am := e.futureMarkets[m.Mkt().ID]
		m.SetNextMTM(am.GetNextMTM())
		e.log.Debug("serialising market", logging.String("id", m.Mkt().ID))
		mks = append(mks, m.GetState())

		if _, ok := e.generatedProviders[m.GetID()]; !ok {
			e.newGeneratedProviders = append(e.newGeneratedProviders, m.GetNewStateProviders()...)
			e.generatedProviders[m.GetID()] = struct{}{}
		}
	}

	return mks, e.newGeneratedProviders
}

func (e *Engine) spotMarketsStates() ([]*types.ExecSpotMarket, []types.StateProvider) {
	mkts := len(e.spotMarketsCpy)
	if mkts == 0 {
		return nil, nil
	}
	mks := make([]*types.ExecSpotMarket, 0, mkts)

	// we don't really know how many new markets there are so don't bother with the calculation
	e.newGeneratedProviders = []types.StateProvider{}
	for _, m := range e.spotMarketsCpy {
		e.log.Debug("serialising spot market", logging.String("id", m.Mkt().ID))
		mks = append(mks, m.GetState())
		if _, ok := e.generatedProviders[m.GetID()]; !ok {
			e.newGeneratedProviders = append(e.newGeneratedProviders, m.GetNewStateProviders()...)
			e.generatedProviders[m.GetID()] = struct{}{}
		}
	}

	return mks, e.newGeneratedProviders
}

func (e *Engine) restoreSpotMarket(ctx context.Context, em *types.ExecSpotMarket) (*spot.Market, error) {
	marketConfig := em.Market
	if len(marketConfig.ID) == 0 {
		return nil, ErrNoMarketID
	}

	// ensure the asset for this new market exists
	asts, err := marketConfig.GetAssets()
	if err != nil {
		return nil, err
	}

	assetDetatils := []*assets.Asset{}
	for _, asset := range asts {
		if !e.collateral.AssetExists(asset) {
			return nil, fmt.Errorf(
				"unable to restore a spot market %q with an invalid %q asset",
				marketConfig.ID,
				asset,
			)
		}
		ad, err := e.assets.Get(asset)
		if err != nil {
			e.log.Error("Failed to restore a market, unknown asset",
				logging.MarketID(marketConfig.ID),
				logging.String("asset-id", asset),
				logging.Error(err),
			)
			return nil, err
		}
		assetDetatils = append(assetDetatils, ad)
	}

	nextMTM := time.Unix(0, em.NextMTM)
	// create market auction state
	e.log.Info("restoring market", logging.String("id", em.Market.ID))

	mkt, err := spot.NewMarketFromSnapshot(
		ctx,
		e.log,
		em,
		e.Config.Risk,
		e.Config.Position,
		e.Config.Settlement,
		e.Config.Matching,
		e.Config.Fee,
		e.Config.Liquidity,
		e.collateral,
		e.oracle,
		e.timeService,
		e.broker,
		e.stateVarEngine,
		assetDetatils[0],
		assetDetatils[1],
		e.marketActivityTracker,
		e.peggedOrderCountUpdated,
		e.referralDiscountRewardService,
		e.volumeDiscountService,
		e.volumeRebateService,
		e.banking,
	)
	if err != nil {
		e.log.Error("failed to instantiate market",
			logging.MarketID(marketConfig.ID),
			logging.Error(err),
		)
		return nil, err
	}
	e.delayTransactionsTarget.MarketDelayRequiredUpdated(mkt.GetID(), marketConfig.EnableTxReordering)
	e.spotMarkets[marketConfig.ID] = mkt
	e.spotMarketsCpy = append(e.spotMarketsCpy, mkt)
	e.allMarkets[marketConfig.ID] = mkt

	if err := e.propagateSpotInitialNetParams(ctx, mkt, true); err != nil {
		return nil, err
	}
	// ensure this is set correctly
	mkt.SetNextMTM(nextMTM)
	return mkt, nil
}

func (e *Engine) restoreMarket(ctx context.Context, em *types.ExecMarket) (*future.Market, error) {
	marketConfig := em.Market

	if len(marketConfig.ID) == 0 {
		return nil, ErrNoMarketID
	}

	// ensure the asset for this new market exists
	assets, err := marketConfig.GetAssets()
	if err != nil {
		return nil, err
	}
	asset := assets[0]
	if !e.collateral.AssetExists(asset) {
		return nil, fmt.Errorf(
			"unable to create a market %q with an invalid %q asset",
			marketConfig.ID,
			asset,
		)
	}
	ad, err := e.assets.Get(asset)
	if err != nil {
		e.log.Error("Failed to restore a market, unknown asset",
			logging.MarketID(marketConfig.ID),
			logging.String("asset-id", asset),
			logging.Error(err),
		)
		return nil, err
	}

	nextMTM := time.Unix(0, em.NextMTM)
	nextInternalCompositePriceCalc := time.Unix(0, em.NextInternalCompositePriceCalc)

	// create market auction state
	e.log.Info("restoring market", logging.String("id", em.Market.ID))
	mkt, err := future.NewMarketFromSnapshot(
		ctx,
		e.log,
		em,
		e.Config.Risk,
		e.Config.Position,
		e.Config.Settlement,
		e.Config.Matching,
		e.Config.Fee,
		e.Config.Liquidity,
		e.collateral,
		e.oracle,
		e.timeService,
		e.broker,
		e.stateVarEngine,
		ad,
		e.marketActivityTracker,
		e.peggedOrderCountUpdated,
		e.referralDiscountRewardService,
		e.volumeDiscountService,
		e.volumeRebateService,
		e.banking,
		e.parties,
	)
	if err != nil {
		e.log.Error("failed to instantiate market",
			logging.MarketID(marketConfig.ID),
			logging.Error(err),
		)
		return nil, err
	}
	if em.IsSucceeded {
		mkt.SetSucceeded()
	}

	e.delayTransactionsTarget.MarketDelayRequiredUpdated(mkt.GetID(), marketConfig.EnableTxReordering)
	e.futureMarkets[marketConfig.ID] = mkt
	e.futureMarketsCpy = append(e.futureMarketsCpy, mkt)
	e.allMarkets[marketConfig.ID] = mkt

	if err := e.propagateInitialNetParamsToFutureMarket(ctx, mkt, true); err != nil {
		return nil, err
	}
	// ensure this is set correctly
	mkt.SetNextMTM(nextMTM)
	mkt.SetNextInternalCompositePriceCalc(nextInternalCompositePriceCalc)
	return mkt, nil
}

func (e *Engine) restoreMarketsStates(ctx context.Context, ems []*types.ExecMarket) ([]types.StateProvider, error) {
	e.futureMarkets = map[string]*future.Market{}

	pvds := make([]types.StateProvider, 0, len(ems)*4)
	for _, em := range ems {
		m, err := e.restoreMarket(ctx, em)
		if err != nil {
			return nil, fmt.Errorf("failed to restore market: %w", err)
		}

		pvds = append(pvds, m.GetNewStateProviders()...)

		// so that we don't return them again the next state change
		e.generatedProviders[m.GetID()] = struct{}{}
	}

	return pvds, nil
}

func (e *Engine) restoreSpotMarketsStates(ctx context.Context, ems []*types.ExecSpotMarket) ([]types.StateProvider, error) {
	e.spotMarkets = map[string]*spot.Market{}

	pvds := make([]types.StateProvider, 0, len(ems)*4)
	for _, em := range ems {
		m, err := e.restoreSpotMarket(ctx, em)
		if err != nil {
			return nil, fmt.Errorf("failed to restore spot market: %w", err)
		}

		pvds = append(pvds, m.GetNewStateProviders()...)

		// so that we don't return them again the next state change
		e.generatedProviders[m.GetID()] = struct{}{}
	}

	return pvds, nil
}

func (e *Engine) serialise() (snapshot []byte, providers []types.StateProvider, err error) {
	mkts, pvds := e.marketsStates()
	cpStates := make([]*types.CPMarketState, 0, len(e.marketCPStates))
	for _, cp := range e.marketCPStates {
		if cp.Market != nil {
			cpy := cp
			cpStates = append(cpStates, cpy)
		}
	}
	// ensure the states are sorted
	sort.SliceStable(cpStates, func(i, j int) bool {
		return cpStates[i].Market.ID > cpStates[j].Market.ID
	})
	successors := make([]*types.Successors, 0, len(e.successors))
	for pid, ids := range e.successors {
		if _, ok := e.GetMarket(pid, true); !ok {
			continue
		}
		successors = append(successors, &types.Successors{
			ParentMarket:     pid,
			SuccessorMarkets: ids,
		})
	}
	sort.SliceStable(successors, func(i, j int) bool {
		return successors[i].ParentMarket > successors[j].ParentMarket
	})

	spotMkts, spotPvds := e.spotMarketsStates()

	allMarketIDs := make([]string, 0, len(e.allMarketsCpy))
	for _, cm := range e.allMarketsCpy {
		allMarketIDs = append(allMarketIDs, cm.GetID())
	}

	tags := e.communityTags.serialize()

	pl := types.Payload{
		Data: &types.PayloadExecutionMarkets{
			ExecutionMarkets: &types.ExecutionMarkets{
				Markets:             mkts,
				SpotMarkets:         spotMkts,
				SettledMarkets:      cpStates,
				Successors:          successors,
				AllMarketIDs:        allMarketIDs,
				MarketCommunityTags: tags,
			},
		},
	}

	s, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return nil, nil, err
	}
	e.snapshotSerialised = s

	return s, append(pvds, spotPvds...), nil
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.ExecutionSnapshot
}

func (e *Engine) Keys() []string {
	return []string{marketsKey}
}

func (e *Engine) Stopped() bool {
	return false
}

func (e *Engine) GetState(_ string) ([]byte, []types.StateProvider, error) {
	serialised, providers, err := e.serialise()
	if err != nil {
		return nil, providers, err
	}

	return serialised, providers, nil
}

func (e *Engine) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	switch pl := payload.Data.(type) {
	case *types.PayloadExecutionMarkets:
		providers, err := e.restoreMarketsStates(ctx, pl.ExecutionMarkets.Markets)
		if err != nil {
			return nil, fmt.Errorf("failed to restore markets states: %w", err)
		}
		// restore settled market state
		for _, m := range pl.ExecutionMarkets.SettledMarkets {
			cpy := m
			e.marketCPStates[m.Market.ID] = cpy
		}
		e.restoreSuccessorMaps(pl.ExecutionMarkets.Successors)
		e.snapshotSerialised, err = proto.Marshal(payload.IntoProto())
		if err != nil {
			return nil, err
		}
		spotProviders, err := e.restoreSpotMarketsStates(ctx, pl.ExecutionMarkets.SpotMarkets)
		e.allMarketsCpy = make([]common.CommonMarket, 0, len(e.allMarkets))
		for _, v := range pl.ExecutionMarkets.AllMarketIDs {
			if mkt, ok := e.allMarkets[v]; ok {
				e.allMarketsCpy = append(e.allMarketsCpy, mkt)
			}
		}

		e.communityTags = NewMarketCommunityTagFromProto(e.broker, pl.ExecutionMarkets.MarketCommunityTags)

		return append(providers, spotProviders...), err
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreSuccessorMaps(successors []*types.Successors) {
	for _, suc := range successors {
		e.successors[suc.ParentMarket] = suc.SuccessorMarkets
		for _, s := range suc.SuccessorMarkets {
			e.isSuccessor[s] = suc.ParentMarket
		}
	}
}

func (e *Engine) OnStateLoaded(ctx context.Context) error {
	for _, m := range e.allMarkets {
		if err := m.PostRestore(ctx); err != nil {
			return err
		}

		// let the core state API know about the market, but not the market data since that will get sent on the very next tick
		// if we sent it now it will create market-data from the core state at the *end* of the block when it was originally
		// created from core-state at the *start* of the block.
		e.broker.Send(events.NewMarketCreatedEvent(ctx, *m.Mkt()))
	}
	// use the time as restored by the snapshot
	t := e.timeService.GetTimeNow()
	// restore marketCPStates through marketsCpy to ensure the order is preserved
	for _, m := range e.futureMarketsCpy {
		if !m.IsSucceeded() {
			cps := m.GetCPState()
			cps.TTL = t.Add(e.successorWindow)
			e.marketCPStates[m.GetID()] = cps
		}
	}
	return nil
}
