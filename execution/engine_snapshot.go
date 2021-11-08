package execution

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

var marketsKey = (&types.PayloadExecutionMarkets{}).Key()

func (e *Engine) marketsStates() ([]*types.ExecMarket, []types.StateProvider, error) {
	mks := make([]*types.ExecMarket, 0, len(e.marketsCpy))
	e.marketsStateProviders = make([]types.StateProvider, 0, (len(e.marketsCpy)-len(e.previouslySnapshottedMarkets))*4)
	for _, m := range e.marketsCpy {
		mks = append(mks, m.getState())

		if _, ok := e.previouslySnapshottedMarkets[m.GetID()]; !ok {
			e.marketsStateProviders = append(e.marketsStateProviders, m.position, m.matching, m.tsCalc, m.liquidity)
			e.previouslySnapshottedMarkets[m.GetID()] = struct{}{}
		}
	}

	return mks, e.marketsStateProviders, nil
}

func (e *Engine) restoreMarket(ctx context.Context, em *types.ExecMarket) (*Market, error) {
	marketConfig := em.Market

	if len(marketConfig.ID) == 0 {
		return nil, ErrNoMarketID
	}
	now := e.time.GetTimeNow()

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

	// create market auction state
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
		now,
		e.broker,
		e.idgen,
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
	}

	return pvds, nil
}

func (e *Engine) getSerialiseSnapshotAndHash() (snapshot, hash []byte, providers []types.StateProvider, err error) {
	if !e.changed() {
		return e.snapshotSerialised, e.snapshotHash, e.marketsStateProviders, nil
	}

	mkts, pvds, err := e.marketsStates()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get market states: %w", err)
	}

	pl := types.Payload{
		Data: &types.PayloadExecutionMarkets{
			ExecutionMarkets: &types.ExecutionMarkets{
				Markets:   mkts,
				Batches:   e.idgen.batches,
				Orders:    e.idgen.orders,
				Proposals: e.idgen.proposals,
			},
		},
	}

	s, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return nil, nil, nil, err
	}

	h := crypto.Hash(s)

	e.snapshotSerialised = s
	e.snapshotHash = h

	return s, h, pvds, nil
}

func (e *Engine) changed() bool {
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

func (e *Engine) GetHash(_ string) ([]byte, error) {
	_, hash, _, err := e.getSerialiseSnapshotAndHash()
	if err != nil {
		return nil, err
	}

	return hash, nil
}

func (e *Engine) GetState(_ string) ([]byte, []types.StateProvider, error) {
	serialised, _, providers, err := e.getSerialiseSnapshotAndHash()
	if err != nil {
		return nil, providers, err
	}

	return serialised, providers, nil
}

func (e *Engine) restoreIDGenerator(em *types.ExecutionMarkets) {
	e.idgen.batches = em.Batches
	e.idgen.proposals = em.Proposals
	e.idgen.orders = em.Orders
}

func (e *Engine) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	switch pl := payload.Data.(type) {
	case *types.PayloadExecutionMarkets:

		e.restoreIDGenerator(pl.ExecutionMarkets)

		providers, err := e.restoreMarketsStates(ctx, pl.ExecutionMarkets.Markets)
		if err != nil {
			return nil, fmt.Errorf("failed to restore markets states: %w", err)
		}

		return providers, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}
