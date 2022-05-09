package statevar

import (
	"context"
	"errors"
	"sort"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	key                        = (&types.PayloadFloatingPointConsensus{}).Key()
	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for floating point consensus snapshot")
	hashKeys                   = []string{key}
)

type snapshotState struct {
	changed    bool
	serialised []byte
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.FloatingPointConsensusSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

func (e *Engine) serialiseNextTimeTrigger() []*snapshot.NextTimeTrigger {
	e.log.Debug("serialising statevar snapshot", logging.Int("n_triggers", len(e.stateVarToNextCalc)))
	timeTriggers := make([]*snapshot.NextTimeTrigger, 0, len(e.stateVarToNextCalc))

	ids := make([]string, 0, len(e.stateVarToNextCalc))
	for id := range e.stateVarToNextCalc {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		if sv, ok := e.stateVars[id]; ok {
			data := &snapshot.NextTimeTrigger{
				Asset:       sv.asset,
				Market:      sv.market,
				Id:          id,
				NextTrigger: e.stateVarToNextCalc[id].UnixNano(),
			}
			timeTriggers = append(timeTriggers, data)
		}
	}

	return timeTriggers
}

// get the serialised form of the given key.
func (e *Engine) serialise(k string) ([]byte, error) {
	if k != key {
		return nil, ErrSnapshotKeyDoesNotExist
	}

	if !e.ss.changed {
		return e.ss.serialised, nil
	}

	payload := types.Payload{
		Data: &types.PayloadFloatingPointConsensus{
			ConsensusData: e.serialiseNextTimeTrigger(),
		},
	}
	data, err := proto.Marshal(payload.IntoProto())
	if err != nil {
		return nil, err
	}

	e.ss.serialised = data
	e.ss.changed = false
	return data, nil
}

func (e *Engine) HasChanged(k string) bool {
	return e.ss.changed
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadFloatingPointConsensus:
		return nil, e.restore(ctx, pl.ConsensusData, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restore(ctx context.Context, nextTimeTrigger []*snapshot.NextTimeTrigger, p *types.Payload) error {
	e.log.Debug("restoring statevar snapshot", logging.Int("n_triggers", len(nextTimeTrigger)))
	for _, data := range nextTimeTrigger {
		e.readyForTimeTrigger[data.Asset+data.Market] = struct{}{}
		e.stateVarToNextCalc[data.Id] = time.Unix(0, data.NextTrigger)
		e.log.Debug("restoring", logging.String("id", data.Id), logging.Time("time", time.Unix(0, data.NextTrigger)))
	}
	var err error
	e.ss.changed = false
	e.ss.serialised, err = proto.Marshal(p.IntoProto())
	return err
}
