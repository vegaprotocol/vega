package spam

import (
	"context"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.SpamSnapshot
}

func (e *Engine) Keys() []string {
	return e.hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

// get the serialised form and hash of the given key.
func (e *Engine) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if _, ok := e.policyNameToPolicy[k]; !ok {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	data, err := e.policyNameToPolicy[k].Serialise()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	return data, hash, nil
}

func (e *Engine) GetHash(k string) ([]byte, error) {
	_, hash, err := e.getSerialisedAndHash(k)
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, _, err := e.getSerialisedAndHash(k)
	return state, nil, err
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	if _, ok := e.policyNameToPolicy[p.Key()]; !ok {
		return nil, types.ErrUnknownSnapshotType
	}

	return nil, e.policyNameToPolicy[p.Key()].Deserialise(p)
}

// OnEpochEvent is a callback for epoch events.
func (e *Engine) OnEpochRestore(ctx context.Context, epoch types.Epoch) {
	e.log.Debug("epoch restoration notification received", logging.String("epoch", epoch.String()))
	e.currentEpoch = &epoch
}
