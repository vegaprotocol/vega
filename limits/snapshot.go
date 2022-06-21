package limits

import (
	"context"

	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	allKey = (&types.PayloadLimitState{}).Key()

	hashKeys = []string{
		allKey,
	}
)

type limitsSnapshotState struct {
	serialised []byte
	changed    bool
}

// serialiseLimits returns the engine's limit data as marshalled bytes.
func (e *Engine) serialiseLimits() ([]byte, error) {
	pl := types.Payload{
		Data: &types.PayloadLimitState{
			LimitState: &types.LimitState{
				BlockCount:               uint32(e.blockCount),
				CanProposeMarket:         e.canProposeMarket,
				CanProposeAsset:          e.canProposeAsset,
				GenesisLoaded:            e.genesisLoaded,
				ProposeMarketEnabled:     e.proposeMarketEnabled,
				ProposeAssetEnabled:      e.proposeAssetEnabled,
				ProposeMarketEnabledFrom: e.proposeMarketEnabledFrom,
				ProposeAssetEnabledFrom:  e.proposeAssetEnabledFrom,
			},
		},
	}
	return proto.Marshal(pl.IntoProto())
}

// get the serialised form of the given key.
func (e *Engine) serialise(k string) ([]byte, error) {
	if k != allKey {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !e.HasChanged(k) {
		return e.lss.serialised, nil
	}

	data, err := e.serialiseLimits()
	if err != nil {
		return nil, err
	}

	e.lss.serialised = data
	e.lss.changed = false
	return data, nil
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.LimitSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

func (e *Engine) HasChanged(k string) bool {
	// return e.lss.changed
	return true
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := e.serialise(k)
	return data, nil, err
}

func (e *Engine) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadLimitState:
		return nil, e.restoreLimits(ctx, pl.LimitState, payload)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreLimits(ctx context.Context, l *types.LimitState, p *types.Payload) error {
	e.blockCount = uint16(l.BlockCount)
	e.canProposeAsset = l.CanProposeAsset
	e.canProposeMarket = l.CanProposeMarket
	e.genesisLoaded = l.GenesisLoaded
	e.proposeMarketEnabled = l.ProposeMarketEnabled
	e.proposeAssetEnabled = l.ProposeAssetEnabled
	e.proposeMarketEnabledFrom = l.ProposeMarketEnabledFrom
	e.proposeAssetEnabledFrom = l.ProposeAssetEnabledFrom

	if e.blockCount > e.bootstrapBlockCount {
		e.bootstrapFinished = true
	}

	e.sendEvent(ctx)
	var err error
	e.lss.changed = false
	e.lss.serialised, err = proto.Marshal(p.IntoProto())
	return err
}
