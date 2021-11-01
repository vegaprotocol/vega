package oracles

import (
	"context"
	"errors"
	"sort"

	"github.com/golang/protobuf/proto"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
)

var (
	key = (&types.PayloadOracleData{}).Key()

	hashKeys = []string{
		key,
	}

	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for oracle data snapshot")
)

type odSnapshotState struct {
	changed    bool
	hash       []byte
	serialised []byte
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.OracleDataSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) dataMapToSlice(m map[string]string) []*types.OracleDataPair {
	odp := make([]*types.OracleDataPair, 0, len(m))
	for k, v := range m {
		odp = append(odp, &types.OracleDataPair{Key: k, Value: v})
	}

	sort.SliceStable(odp, func(i, j int) bool { return odp[i].Key < odp[j].Key })
	return odp
}

func (e *Engine) sliceToMap(ods []*types.OracleDataPair) map[string]string {
	m := make(map[string]string, len(ods))
	for _, od := range ods {
		m[od.Key] = od.Value
	}
	return m
}

func (e *Engine) serialise() ([]byte, error) {
	oracleData := make([]*types.OracleData, 0, len(e.buffer))
	for _, b := range e.buffer {
		oracleData = append(oracleData, &types.OracleData{
			PubKeys: b.PubKeys,
			Data:    e.dataMapToSlice(b.Data),
		})
	}

	payload := types.Payload{
		Data: &types.PayloadOracleData{
			OracleData: oracleData,
		},
	}
	x := payload.IntoProto()
	return proto.Marshal(x)
}

// get the serialised form and hash of the given key.
func (e *Engine) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != key {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !e.odss.changed {
		return e.odss.serialised, e.odss.hash, nil
	}

	data, err := e.serialise()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	e.odss.serialised = data
	e.odss.hash = hash
	e.odss.changed = false
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
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadOracleData:
		return nil, e.restore(ctx, pl.OracleData)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restore(ctx context.Context, oracleData []*types.OracleData) error {
	e.buffer = make([]OracleData, 0, len(oracleData))
	for _, od := range oracleData {
		e.buffer = append(e.buffer, OracleData{
			PubKeys: od.PubKeys,
			Data:    e.sliceToMap(od.Data),
		})
	}

	e.odss.changed = true
	return nil
}
