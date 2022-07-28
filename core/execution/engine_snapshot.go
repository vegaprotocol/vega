// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package execution

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/core/types"
)

var marketsKey = (&types.PayloadExecutionMarkets{}).Key()

func (e *Engine) marketsStates() ([]*types.ExecMarket, []types.StateProvider, error) {
	mkts := len(e.marketsCpy)
	if mkts == 0 {
		return nil, nil, nil
	}
	mks := make([]*types.ExecMarket, 0, mkts)
	if prev := len(e.generatedProviders); prev < mkts {
		mkts -= prev
	}
	e.newGeneratedProviders = make([]types.StateProvider, 0, mkts*4)
	for _, m := range e.marketsCpy {
		e.log.Debug("serialising market", logging.String("id", m.mkt.ID))
		mks = append(mks, m.getState())

		if _, ok := e.generatedProviders[m.GetID()]; !ok {
			e.newGeneratedProviders = append(e.newGeneratedProviders, m.position, m.matching, m.tsCalc, m.liquidity)
			e.generatedProviders[m.GetID()] = struct{}{}
		}
	}

	return mks, e.newGeneratedProviders, nil
}

func (e *Engine) restoreMarket(ctx context.Context, em *types.ExecMarket) (*Market, error) {
	marketConfig := em.Market

	if len(marketConfig.ID) == 0 {
		return nil, ErrNoMarketID
	}

	// ensure the asset for this new market exists
	asset, err := marketConfig.GetAsset()
	if err != nil {
		return nil, err
	}
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

	// create market auction state
	e.log.Info("restoring market", logging.String("id", em.Market.ID))
	mkt, err := NewMarketFromSnapshot(
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
	)
	if err != nil {
		e.log.Error("failed to instantiate market",
			logging.MarketID(marketConfig.ID),
			logging.Error(err),
		)
		return nil, err
	}

	e.markets[marketConfig.ID] = mkt
	e.marketsCpy = append(e.marketsCpy, mkt)

	if err := e.propagateInitialNetParams(ctx, mkt); err != nil {
		return nil, err
	}

	e.publishNewMarketInfos(ctx, mkt)
	return mkt, nil
}

func (e *Engine) restoreMarketsStates(ctx context.Context, ems []*types.ExecMarket) ([]types.StateProvider, error) {
	e.markets = map[string]*Market{}

	pvds := make([]types.StateProvider, 0, len(ems)*4)
	for _, em := range ems {
		m, err := e.restoreMarket(ctx, em)
		if err != nil {
			return nil, fmt.Errorf("failed to restore market: %w", err)
		}

		pvds = append(pvds, m.position, m.matching, m.tsCalc, m.liquidity)

		// so that we don't return them again the next state change
		e.generatedProviders[m.GetID()] = struct{}{}
	}

	return pvds, nil
}

func (e *Engine) serialise() (snapshot []byte, providers []types.StateProvider, err error) {
	if !e.HasChanged("") {
		return e.snapshotSerialised, e.newGeneratedProviders, nil
	}

	mkts, pvds, err := e.marketsStates()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get market states: %w", err)
	}

	pl := types.Payload{
		Data: &types.PayloadExecutionMarkets{
			ExecutionMarkets: &types.ExecutionMarkets{
				Markets: mkts,
			},
		},
	}

	s, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return nil, nil, err
	}
	e.snapshotSerialised = s
	e.stateChanged = false

	return s, pvds, nil
}

func (e *Engine) changed() bool {
	if len(e.snapshotSerialised) == 0 {
		return true
	}

	if e.stateChanged {
		e.log.Debug("state-changed in execution engine itself")
		return true
	}
	for _, m := range e.markets {
		if m.changed() {
			return true
		}
	}

	return false
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

func (e *Engine) HasChanged(k string) bool {
	return true
	// return e.changed()
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
		e.snapshotSerialised, err = proto.Marshal(payload.IntoProto())
		e.stateChanged = false
		return providers, err
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) OnStateLoaded(ctx context.Context) error {
	for _, m := range e.markets {
		if err := m.PostRestore(ctx); err != nil {
			return err
		}
	}
	return nil
}
