package liquidation

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
)

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.LiquidationSnapshot
}

func (e *Engine) Keys() []string {
	return []string{e.mID}
}

// GetState must be thread-safe as it may be called from multiple goroutines concurrently!
func (e *Engine) GetState(key string) ([]byte, []types.StateProvider, error) {
	if key != e.mID {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}
	if e.stopped {
		return nil, nil, nil
	}
	payload := e.buildPayload()

	s, err := proto.Marshal(payload.IntoProto())
	return s, nil, err
}

func (e *Engine) LoadState(ctx context.Context, pl *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != pl.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch d := pl.Data.(type) {
	case *types.LiquidationNode:
		e.mID = d.MarketID
		e.pos.open = d.NetworkPos
		e.nextStep = d.NextStep
		if d.Config != nil {
			e.cfg = d.Config.DeepClone()
		} else {
			// @NOTE this can be removed after protocol upgrade has completed
			e.cfg = GetLegacyStrat()
		}
	default:
		return nil, types.ErrUnknownSnapshotType
	}
	return nil, nil
}

func (e *Engine) Stopped() bool {
	return e.stopped
}

func (e *Engine) StopSnapshots() {
	e.stopped = true
}

func (e *Engine) buildPayload() *types.Payload {
	// this should not be needed
	var cfg *types.LiquidationStrategy
	if e.cfg != nil {
		cfg = e.cfg.DeepClone()
	}
	return &types.Payload{
		Data: &types.LiquidationNode{
			MarketID:   e.mID,
			NetworkPos: e.pos.open,
			NextStep:   e.nextStep,
			Config:     cfg,
		},
	}
}
