package spam

import (
	"context"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
)

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.SpamSnapshot
}

func (e *Engine) Keys() []string {
	return e.hashKeys
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

func (e *Engine) GetState(k string) ([]byte, error) {
	state, _, err := e.getSerialisedAndHash(k)
	return state, err
}

func (e *Engine) Snapshot() (map[string][]byte, error) {
	r := make(map[string][]byte, len(e.hashKeys))
	for _, k := range e.hashKeys {
		state, err := e.GetState(k)
		if err != nil {
			return nil, err
		}
		r[k] = state
	}
	return r, nil
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) error {
	if e.Namespace() != p.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	if _, ok := e.policyNameToPolicy[p.Key()]; !ok {
		return types.ErrUnknownSnapshotType
	}

	return e.policyNameToPolicy[p.Key()].Deserialise(p)
}
