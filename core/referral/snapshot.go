// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package referral

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/maps"
)

type SnapshottedEngine struct {
	*Engine

	pl types.Payload

	stopped bool

	// Keys need to be computed when the engine is instantiated as they are dynamic.
	hashKeys []string
	key      string
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
	case *types.PayloadReferralProgramState:
		e.load(data)
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
	case e.key:
		return e.serialiseReferralProgram()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *SnapshottedEngine) serialiseReferralProgram() ([]byte, error) {
	referralProgramData := &snapshotpb.ReferralProgramData{
		LastProgramVersion: e.latestProgramVersion,
		ProgramHasEnded:    e.programHasEnded,
	}

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_ReferralProgram{
			ReferralProgram: referralProgramData,
		},
	}

	if e.currentProgram != nil {
		referralProgramData.CurrentProgram = e.currentProgram.IntoProto()
	}
	if e.newProgram != nil {
		referralProgramData.NewProgram = e.newProgram.IntoProto()
	}

	referralProgramData.FactorByReferee = make([]*snapshotpb.FactorByReferee, 0, len(e.factorsByReferee))
	for pi, rs := range e.factorsByReferee {
		tv := rs.TakerVolume.Bytes()
		referralProgramData.FactorByReferee = append(referralProgramData.FactorByReferee, &snapshotpb.FactorByReferee{
			Party: pi.String(), DiscountFactors: rs.DiscountFactors.IntoDiscountFactorsProto(), TakerVolume: tv[:],
		})
	}

	sort.Slice(referralProgramData.FactorByReferee, func(i, j int) bool {
		return referralProgramData.FactorByReferee[i].Party < referralProgramData.FactorByReferee[j].Party
	})

	referralProgramData.Sets = make([]*snapshotpb.ReferralSet, 0, len(e.sets))
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
			CurrentRewardFactors:            set.CurrentRewardFactors.IntoRewardFactorsProto(),
			CurrentRewardsMultiplier:        set.CurrentRewardsMultiplier.String(),
			CurrentRewardsFactorsMultiplier: set.CurrentRewardsFactorMultiplier.IntoRewardFactorsProto(),
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
				var b []byte
				if volume != nil {
					bb := volume.value.Bytes()
					b = bb[:]
				}
				runningVolumesProto = append(runningVolumesProto, &snapshotpb.RunningVolume{
					Epoch:  volume.epoch,
					Volume: b,
				})
			}
			setProto.RunningVolumes = runningVolumesProto
		}

		referralProgramData.Sets = append(referralProgramData.Sets, setProto)
	}

	serialised, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize referral misc payload: %w", err)
	}

	return serialised, nil
}

func (e *SnapshottedEngine) buildHashKeys() {
	e.key = (&types.PayloadReferralProgramState{}).Key()
	e.hashKeys = append([]string{}, e.key)
}

func NewSnapshottedEngine(broker Broker, timeSvc TimeService, mat MarketActivityTracker, staking StakingBalances) *SnapshottedEngine {
	se := &SnapshottedEngine{
		Engine:  NewEngine(broker, timeSvc, mat, staking),
		pl:      types.Payload{},
		stopped: false,
	}

	se.buildHashKeys()

	return se
}
