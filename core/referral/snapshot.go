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
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"golang.org/x/exp/maps"
)

type SnapshottedEngine struct {
	*Engine

	pl types.Payload

	stopped bool

	// Keys need to be computed when the engine is instantiated as they are dynamic.
	hashKeys          []string
	currentProgramKey string
	newProgramKey     string
	referralSetsKey   string
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
		e.loadCurrentReferralProgramFromSnapshot(data.CurrentReferralProgram)
		return nil, nil
	case *types.PayloadNewReferralProgram:
		e.loadNewReferralProgramFromSnapshot(data.NewReferralProgram)
		return nil, nil
	case *types.PayloadReferralSets:
		e.loadReferralSetsFromSnapshot(data.Sets)
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
	case e.referralSetsKey:
		return e.serialiseReferralSets()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *SnapshottedEngine) serialiseReferralSets() ([]byte, error) {
	setsProto := make([]*snapshotpb.ReferralSet, 0, len(e.sets))

	setIDs := maps.Keys(e.sets)

	sort.SliceStable(setIDs, func(i, j int) bool {
		return setIDs[i] < setIDs[j]
	})

	for _, setID := range setIDs {
		set := e.sets[setID]
		setProto := &snapshotpb.ReferralSet{
			Id:        string(set.ID),
			CreatedAt: set.CreatedAt.UnixNano(),
			UpdatedAt: set.UpdatedAt.UnixNano(),
			Referrer: &snapshotpb.Membership{
				PartyId:        string(set.Referrer.PartyID),
				JoinedAt:       set.Referrer.JoinedAt.UnixNano(),
				StartedAtEpoch: set.Referrer.StartedAtEpoch,
			},
		}

		for _, r := range set.Referees {
			setProto.Referees = append(setProto.Referees,
				&snapshotpb.Membership{
					PartyId:        string(r.PartyID),
					JoinedAt:       r.JoinedAt.UnixNano(),
					StartedAtEpoch: r.StartedAtEpoch,
				},
			)
		}

		runningVolumes, isTracked := e.referralSetsNotionalVolumes.runningVolumesBySet[set.ID]
		if isTracked {
			runningVolumesProto := make([]*snapshotpb.RunningVolume, 0, len(runningVolumes))
			for _, volume := range runningVolumes {
				runningVolumesProto = append(runningVolumesProto, &snapshotpb.RunningVolume{
					Epoch:  volume.epoch,
					Volume: volume.value.String(),
				})
			}
			setProto.RunningVolumes = runningVolumesProto
		}

		setsProto = append(setsProto, setProto)
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_ReferralSets{
			ReferralSets: &snapshotpb.ReferralSets{
				Sets: setsProto,
			},
		},
	}

	serialisedSets, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize referral sets payload: %w", err)
	}

	return serialisedSets, nil
}

func (e *SnapshottedEngine) serialiseCurrentReferralProgram() ([]byte, error) {
	var programSnapshot *vegapb.ReferralProgram
	if e.currentProgram != nil {
		programSnapshot = e.currentProgram.IntoProto()
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
	if e.newProgram != nil {
		programSnapshot = e.newProgram.IntoProto()
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
	e.referralSetsKey = (&types.PayloadReferralSets{}).Key()

	e.hashKeys = append([]string{}, e.currentProgramKey, e.newProgramKey, e.referralSetsKey)
}

func NewSnapshottedEngine(broker Broker, timeSvc TimeService, mat MarketActivityTracker) *SnapshottedEngine {
	se := &SnapshottedEngine{
		Engine:  NewEngine(broker, timeSvc, mat),
		pl:      types.Payload{},
		stopped: false,
	}

	se.buildHashKeys()

	return se
}
