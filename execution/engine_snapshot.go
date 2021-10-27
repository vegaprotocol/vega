package execution

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

var marketsKey = (&types.PayloadExecutionMarkets{}).Key()

func (e *Engine) sortedMarketIDs() []string {
	ids := make([]string, 0, len(e.markets))
	for id := range e.markets {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	return ids
}

func (e *Engine) marketsStates() ([]*types.ExecMarket, []types.StateProvider, error) {
	// snapshots should be deterministic
	mktIDs := e.sortedMarketIDs()

	mks := make([]*types.ExecMarket, 0, len(mktIDs))
	e.marketsStateProviders = make([]types.StateProvider, 0, (len(mktIDs)-len(e.previouslySnapshottedMarkets))*3)
	for _, id := range mktIDs {
		m, ok := e.markets[id]
		// this should not happen but just in case...
		if !ok {
			return nil, nil, fmt.Errorf("market %q not found in execution engine", id)
		}
		mks = append(mks, m.getState())

		if _, ok := e.previouslySnapshottedMarkets[m.GetID()]; !ok {
			e.marketsStateProviders = append(e.marketsStateProviders, m.position, m.matching)
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
		e.log.Error("unable to create a market with an invalid asset",
			logging.MarketID(marketConfig.ID),
			logging.AssetID(asset))
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

	pvds := make([]types.StateProvider, 0, len(ems)*2)
	for _, em := range ems {
		m, err := e.restoreMarket(ctx, em)
		if err != nil {
			return nil, fmt.Errorf("failed to restore market: %w", err)
		}

		pvds = append(pvds, m.position, m.matching)
	}

	return pvds, nil
}

func (e *Engine) getSerialiseSnapshotAndHash() (snapshot, hash []byte, providers []types.StateProvider, err error) {
	if !e.changed() {
		return e.snapshotSerialised, e.snapshotHash, nil, nil
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
		providers, err := e.restoreMarketsStates(ctx, pl.ExecutionMarkets.Markets)
		if err != nil {
			return nil, fmt.Errorf("failed to restore markets states: %w", err)
		}

		e.restoreIDGenerator(pl.ExecutionMarkets)

		return providers, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}
