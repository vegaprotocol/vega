package limits

import (
	"context"

	"code.vegaprotocol.io/vega/libs/crypto"
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
	hash       []byte
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

// get the serialised form and hash of the given key.
func (e *Engine) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != allKey {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !e.lss.changed {
		return e.lss.serialised, e.lss.hash, nil
	}

	data, err := e.serialiseLimits()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	e.lss.serialised = data
	e.lss.hash = hash
	e.lss.changed = false
	return data, hash, nil
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

func (e *Engine) GetHash(k string) ([]byte, error) {
	_, hash, err := e.getSerialisedAndHash(k)
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, _, err := e.getSerialisedAndHash(k)
	return data, nil, err
}

func (e *Engine) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadLimitState:
		return nil, e.restoreLimits(ctx, pl.LimitState)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreLimits(ctx context.Context, l *types.LimitState) error {
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
	e.lss.changed = true
	return nil
}
