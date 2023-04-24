// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package statevar

import (
	"context"
	"errors"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/logging"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	key                        = (&types.PayloadFloatingPointConsensus{}).Key()
	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for floating point consensus snapshot")
	hashKeys                   = []string{key}
)

type snapshotState struct {
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

func mapToResults(m map[string]*statevar.KeyValueBundle) []*snapshot.FloatingPointValidatorResult {
	if m == nil {
		return []*snapshot.FloatingPointValidatorResult{}
	}
	res := make([]*snapshot.FloatingPointValidatorResult, 0, len(m))
	for k, kvb := range m {
		res = append(res, &snapshot.FloatingPointValidatorResult{Id: k, Bundle: kvb.ToProto()})
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Id < res[j].Id })
	return res
}

func (sv *StateVariable) serialise() *snapshot.StateVarInternalState {
	return &snapshot.StateVarInternalState{
		Id:                          sv.ID,
		EventId:                     sv.eventID,
		State:                       int32(sv.state),
		ValidatorsResults:           mapToResults(sv.validatorResults),
		RoundsSinceMeaningfulUpdate: int32(sv.roundsSinceMeaningfulUpdate),
	}
}

// get the serialised form of the given key.
func (e *Engine) serialise(k string) ([]byte, error) {
	if k != key {
		return nil, ErrSnapshotKeyDoesNotExist
	}

	stateVariablesState := make([]*snapshot.StateVarInternalState, 0, len(e.stateVars))
	for _, sv := range e.stateVars {
		stateVariablesState = append(stateVariablesState, sv.serialise())
	}
	sort.SliceStable(stateVariablesState, func(i, j int) bool { return stateVariablesState[i].Id < stateVariablesState[j].Id })

	payload := types.Payload{
		Data: &types.PayloadFloatingPointConsensus{
			ConsensusData:               e.serialiseNextTimeTrigger(),
			StateVariablesInternalState: stateVariablesState,
		},
	}
	data, err := proto.Marshal(payload.IntoProto())
	if err != nil {
		return nil, err
	}

	e.ss.serialised = data
	return data, nil
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
		return nil, e.restore(pl.ConsensusData, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restore(nextTimeTrigger []*snapshot.NextTimeTrigger, p *types.Payload) error {
	e.log.Debug("restoring statevar snapshot", logging.Int("n_triggers", len(nextTimeTrigger)))
	for _, data := range nextTimeTrigger {
		e.readyForTimeTrigger[data.Asset+data.Market] = struct{}{}
		e.stateVarToNextCalc[data.Id] = time.Unix(0, data.NextTrigger)
		e.log.Debug("restoring", logging.String("id", data.Id), logging.Time("time", time.Unix(0, data.NextTrigger)))
	}
	var err error
	e.ss.serialised, err = proto.Marshal(p.IntoProto())
	return err
}

// postRestore sets the internal state of all state variables from a snapshot. If there is an active event it will initiate the calculation.
func (e *Engine) postRestore(stateVariablesInternalState []*snapshot.StateVarInternalState) {
	for _, svis := range stateVariablesInternalState {
		sv, ok := e.stateVars[svis.Id]
		if !ok {
			e.log.Panic("expecting a state variable with id to exist during post restore", logging.String("ID", svis.Id))
			continue
		}
		sv.eventID = svis.EventId
		sv.state = ConsensusState(svis.State)
		sv.roundsSinceMeaningfulUpdate = uint(svis.RoundsSinceMeaningfulUpdate)
		if len(svis.ValidatorsResults) > 0 {
			sv.validatorResults = make(map[string]*statevar.KeyValueBundle, len(svis.ValidatorsResults))
		}
		for _, fpvr := range svis.ValidatorsResults {
			kvb, err := statevar.KeyValueBundleFromProto(fpvr.Bundle)
			if err != nil {
				e.log.Panic("restoring malformed statevar kvb", logging.String("id", fpvr.Id), logging.Error(err))
			}
			sv.validatorResults[fpvr.Id] = kvb
		}
	}
}

// OnStateLoaded is called after all snapshots have been loaded and hence all state variables have been created and sets the internal state for all state variables.
func (e *Engine) OnStateLoaded(ctx context.Context) error {
	var p snapshot.Payload
	err := proto.Unmarshal(e.ss.serialised, &p)
	if err != nil {
		e.log.Error("failed to deserialise state var payload", logging.String("error", err.Error()))
		return err
	}
	payload := types.PayloadFromProto(&p)
	switch pl := payload.Data.(type) {
	case *types.PayloadFloatingPointConsensus:
		e.postRestore(pl.StateVariablesInternalState)
	}
	return nil
}
