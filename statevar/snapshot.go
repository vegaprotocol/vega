package statevar

import (
	"context"
	"errors"
	"sort"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

var (
	key                        = (&types.PayloadFloatingPointConsensus{}).Key()
	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for floating point consensus snapshot")
	hashKeys                   = []string{key}
)

type snapshotState struct {
	changed    bool
	hash       []byte
	serialised []byte
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.FloatingPointConsensusSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) serialiseNextTimeTrigger() []*snapshot.NextTimeTrigger {
	timeTriggers := make([]*snapshot.NextTimeTrigger, 0, len(e.stateVarToNextCalc))

	ids := make([]string, 0, len(e.stateVarToNextCalc))
	for id := range e.stateVarToNextCalc {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		if sv, ok := e.stateVars[id]; ok {
			timeTriggers = append(timeTriggers, &snapshot.NextTimeTrigger{
				Asset:       sv.asset,
				Market:      sv.market,
				Id:          id,
				NextTrigger: e.stateVarToNextCalc[id].UnixNano(),
			})
		}
	}

	return timeTriggers
}

func (e *Engine) serialise() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadFloatingPointConsensus{
			ConsensusData: e.serialiseNextTimeTrigger(),
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

	if !e.ss.changed {
		return e.ss.serialised, e.ss.hash, nil
	}

	data, err := e.serialise()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	e.ss.serialised = data
	e.ss.hash = hash
	e.ss.changed = false
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
	case *types.PayloadFloatingPointConsensus:
		return nil, e.restore(ctx, pl.ConsensusData)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restore(ctx context.Context, nextTimeTrigger []*snapshot.NextTimeTrigger) error {
	for _, data := range nextTimeTrigger {
		e.readyForTimeTrigger[data.Asset+data.Market] = struct{}{}
		e.stateVarToNextCalc[data.Id] = time.Unix(0, data.NextTrigger)
	}
	e.ss.changed = true
	return nil
}
