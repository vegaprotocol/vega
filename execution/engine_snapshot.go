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

func (e *Engine) marketsStates() []*types.ExecMarket {
	// snapshots should be deterministic
	mktIDs := e.sortedMarketIDs()

	mks := make([]*types.ExecMarket, 0, len(mktIDs))
	for _, id := range mktIDs {
		m, ok := e.markets[id]
		// this should not happen but just in case...
		if !ok {
			continue
		}
		mks = append(mks, m.getState())
	}

	return mks
}

func (e *Engine) restoreMarketsStates(ems []*types.ExecMarket) error {
	for _, em := range ems {
		if _, ok := e.markets[em.Market.ID]; !ok {
			err := e.submitMarket(context.Background(), em.Market.DeepClone())
			if err != nil {
				return err
			}
		}

		m := e.markets[em.Market.ID]
		if err := m.restoreState(em); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) getSerialiseSnapshotAndHash() (snapshot, hash []byte, err error) {
	if !e.changed() {
		return e.snapshotSerialised, e.snapshotHash, nil
	}

	pl := types.Payload{
		Data: &types.PayloadExecutionMarkets{
			ExecutionMarkets: &types.ExecutionMarkets{
				Markets: e.marketsStates(),
			},
		},
	}

	s, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return nil, nil, err
	}

	h := crypto.Hash(s)

	e.snapshotSerialised = s
	e.snapshotHash = h

	return s, h, nil
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
	_, hash, err := e.getSerialiseSnapshotAndHash()
	if err != nil {
		return nil, err
	}

	return hash, nil
}

// Snapshot is a sync call to get the state for all keys.
func (e *Engine) Snapshot() (map[string][]byte, error) {
	serialised, _, err := e.getSerialiseSnapshotAndHash()
	if err != nil {
		return nil, err
	}
	return map[string][]byte{marketsKey: serialised}, nil
}

func (e *Engine) GetState(_ string) ([]byte, error) {
	serialised, _, err := e.getSerialiseSnapshotAndHash()
	if err != nil {
		return nil, err
	}

	return serialised, nil
}

func (e *Engine) LoadState(payload *types.Payload) error {
	switch pl := payload.Data.(type) {
	case *types.PayloadExecutionMarkets:
		if err := e.restoreMarketsStates(pl.ExecutionMarkets.Markets); err != nil {
			return fmt.Errorf("failed to restore markets states: %w", err)
		}
		return nil
	default:
		return types.ErrUnknownSnapshotType
	}
}
