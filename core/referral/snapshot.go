// Copyright (c) 2023 Gobalsky Labs Limited
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

package referral

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type SnapshottedEngine struct {
	*Engine

	pl types.Payload

	stopped bool

	// Keys need to be computed when the engine is instantiated as they are dynamic.
	hashKeys          []string
	currentProgramKey string
	newProgramKey     string
}

func (e *SnapshottedEngine) Namespace() types.SnapshotNamespace {
	return types.ReferralProgramSnapshot
}

func (e *SnapshottedEngine) Keys() []string {
	return e.hashKeys
}

func (e *SnapshottedEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *SnapshottedEngine) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch data := p.Data.(type) {
	case *types.PayloadCurrentReferralProgram:
		e.Engine.loadCurrentReferralProgramFromSnapshot(data.CurrentReferralProgram)
		return nil, nil
	case *types.PayloadNewReferralProgram:
		e.Engine.loadNewReferralProgramFromSnapshot(data.NewReferralProgram)
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshottedEngine) Stopped() bool {
	return e.stopped
}

func (e *SnapshottedEngine) StopSnapshots() {
	e.stopped = true
}

func (e *SnapshottedEngine) serialise(k string) ([]byte, error) {
	if e.stopped {
		return nil, nil
	}

	switch k {
	case e.currentProgramKey:
		return e.serialiseCurrentReferralProgram()
	case e.newProgramKey:
		return e.serialiseNewReferralProgram()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *SnapshottedEngine) serialiseCurrentReferralProgram() ([]byte, error) {
	var programSnapshot *vegapb.ReferralProgram
	if e.Engine.currentProgram != nil {
		programSnapshot = e.Engine.currentProgram.IntoProto()
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_CurrentReferralProgram{
			CurrentReferralProgram: &snapshotpb.CurrentReferralProgram{
				ReferralProgram: programSnapshot,
			},
		},
	}

	serialisedCurrentReferralProgram, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize current referral program payload: %w", err)
	}

	return serialisedCurrentReferralProgram, nil
}

func (e *SnapshottedEngine) serialiseNewReferralProgram() ([]byte, error) {
	var programSnapshot *vegapb.ReferralProgram
	if e.Engine.newProgram != nil {
		programSnapshot = e.Engine.newProgram.IntoProto()
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_NewReferralProgram{
			NewReferralProgram: &snapshotpb.NewReferralProgram{
				ReferralProgram: programSnapshot,
			},
		},
	}

	serialisedNewReferralProgram, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize new referral program payload: %w", err)
	}

	return serialisedNewReferralProgram, nil
}

func (e *SnapshottedEngine) buildHashKeys() {
	e.currentProgramKey = (&types.PayloadCurrentReferralProgram{}).Key()
	e.newProgramKey = (&types.PayloadNewReferralProgram{}).Key()

	e.hashKeys = append([]string{}, e.currentProgramKey, e.newProgramKey)
}

func NewSnapshottedEngine(epochEngine EpochEngine, broker Broker, teamsEngine TeamsEngine) *SnapshottedEngine {
	se := &SnapshottedEngine{
		Engine:  NewEngine(epochEngine, broker, teamsEngine),
		pl:      types.Payload{},
		stopped: false,
	}

	se.buildHashKeys()

	return se
}
