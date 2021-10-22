package execution

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
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

func (e *Engine) marketsStates() ([]*types.ExecMarket, []types.StateProvider) {
	// snapshots should be deterministic
	mktIDs := e.sortedMarketIDs()

	mks := make([]*types.ExecMarket, 0, len(mktIDs))
	e.marketsStateProviders = make([]types.StateProvider, 0, (len(mktIDs)-len(e.previouslySnapshottedMarkets))*2)
	for _, id := range mktIDs {
		m, ok := e.markets[id]
		// this should not happen but just in case...
		if !ok {
			continue
		}
		mks = append(mks, m.getState())

		if _, ok := e.previouslySnapshottedMarkets[m.GetID()]; !ok {
			e.marketsStateProviders = append(e.marketsStateProviders, m.position, m.matching)
			e.previouslySnapshottedMarkets[m.GetID()] = struct{}{}
		}
	}

	return mks, e.marketsStateProviders
}

func (e *Engine) restoreMarketsStates(ems []*types.ExecMarket) ([]types.StateProvider, error) {
	pvds := make([]types.StateProvider, 0, len(ems)*2)

	for _, em := range ems {
		if _, ok := e.markets[em.Market.ID]; !ok {
			err := e.submitMarket(context.Background(), em.Market.DeepClone())
			if err != nil {
				return nil, err
			}
		}

		m := e.markets[em.Market.ID]
		if err := m.restoreState(em); err != nil {
			return nil, err
		}

		pvds = append(pvds, m.position, m.matching)
	}

	return pvds, nil
}

func (e *Engine) getSerialiseSnapshotAndHash() (snapshot, hash []byte, providers []types.StateProvider, err error) {
	if !e.changed() {
		return e.snapshotSerialised, e.snapshotHash, nil, nil
	}

	mkts, pvds := e.marketsStates()

	pl := types.Payload{
		Data: &types.PayloadExecutionMarkets{
			ExecutionMarkets: &types.ExecutionMarkets{
				Markets: mkts,
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

func (e *Engine) LoadState(_ context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	switch pl := payload.Data.(type) {
	case *types.PayloadExecutionMarkets:
		providers, err := e.restoreMarketsStates(pl.ExecutionMarkets.Markets)
		if err != nil {
			return nil, fmt.Errorf("failed to restore markets states: %w", err)
		}
		return providers, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}
