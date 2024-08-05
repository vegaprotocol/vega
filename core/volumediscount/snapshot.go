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

package volumediscount

import (
	"context"
	"sort"
	"strings"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/slices"
)

var (
	key      = (&types.PayloadVolumeDiscountProgram{}).Key()
	hashKeys = []string{key}
)

type SnapshottedEngine struct {
	*Engine

	pl types.Payload
}

func (e *SnapshottedEngine) Namespace() types.SnapshotNamespace {
	return types.VolumeDiscountProgramSnapshot
}

func (e *SnapshottedEngine) Keys() []string {
	return hashKeys
}

func (e *SnapshottedEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *Engine) loadCurrentProgramFromSnapshot(program *vegapb.VolumeDiscountProgram) {
	if program == nil {
		e.currentProgram = nil
		return
	}

	e.currentProgram = types.NewVolumeDiscountProgramFromProto(program)
}

func (e *Engine) loadNewProgramFromSnapshot(program *vegapb.VolumeDiscountProgram) {
	if program == nil {
		e.newProgram = nil
		return
	}

	e.newProgram = types.NewVolumeDiscountProgramFromProto(program)
}

func (e *SnapshottedEngine) restore(vdp *snapshotpb.VolumeDiscountProgram) error {
	e.latestProgramVersion = vdp.LastProgramVersion
	e.programHasEnded = vdp.ProgramHasEnded
	e.loadCurrentProgramFromSnapshot(vdp.CurrentProgram)
	e.loadNewProgramFromSnapshot(vdp.NewProgram)
	for _, v := range vdp.Parties {
		e.parties[types.PartyID(v)] = struct{}{}
	}
	for _, pv := range vdp.AveragePartyVolume {
		volume, err := num.UnmarshalBinaryDecimal(pv.Volume)
		if err != nil {
			return err
		}
		e.avgVolumePerParty[types.PartyID(pv.Party)] = volume
	}
	e.epochDataIndex = int(vdp.EpochDataIndex)
	for i, epv := range vdp.EpochPartyVolumes {
		if len(epv.PartyVolume) > 0 {
			volumes := map[types.PartyID]*num.Uint{}
			for _, pv := range epv.PartyVolume {
				var v *num.Uint
				if len(pv.Volume) > 0 {
					v = num.UintFromBytes(pv.Volume)
				}
				volumes[types.PartyID(pv.Party)] = v
			}
			e.epochData[i] = volumes
		}
	}

	for _, stats := range vdp.FactorsByParty {
		factors := types.FactorsFromDiscountFactorsWithDefault(stats.DiscountFactors, stats.DiscountFactor)
		e.factorsByParty[types.PartyID(stats.Party)] = types.VolumeDiscountStats{
			DiscountFactors: factors,
		}
	}

	return nil
}

func (e *SnapshottedEngine) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch data := p.Data.(type) {
	case *types.PayloadVolumeDiscountProgram:
		return nil, e.restore(data.VolumeDiscountProgram)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshottedEngine) Stopped() bool {
	return false
}

func (e *SnapshottedEngine) serialise(k string) ([]byte, error) {
	switch k {
	case key:
		return e.serialiseDiscountVolumeProgram()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *SnapshottedEngine) serialiseDiscountVolumeProgram() ([]byte, error) {
	parties := make([]string, 0, len(e.parties))
	for pi := range e.parties {
		parties = append(parties, string(pi))
	}
	sort.Strings(parties)

	avgPartyVolumes := make([]*snapshotpb.PartyVolume, 0, len(e.avgVolumePerParty))
	for pi, d := range e.avgVolumePerParty {
		b, _ := d.MarshalBinary()
		avgPartyVolumes = append(avgPartyVolumes, &snapshotpb.PartyVolume{Party: string(pi), Volume: b})
	}
	sort.Slice(avgPartyVolumes, func(i, j int) bool {
		return avgPartyVolumes[i].Party < avgPartyVolumes[j].Party
	})

	epochData := make([]*snapshotpb.EpochPartyVolumes, 0, len(e.epochData))
	for _, epv := range e.epochData {
		ed := &snapshotpb.EpochPartyVolumes{}
		if len(epv) > 0 {
			ed.PartyVolume = make([]*snapshotpb.PartyVolume, 0, len(epv))
			for pi, u := range epv {
				var b []byte
				if u != nil {
					bb := u.Bytes()
					b = bb[:]
				}
				ed.PartyVolume = append(ed.PartyVolume, &snapshotpb.PartyVolume{Party: string(pi), Volume: b})
			}
			sort.Slice(ed.PartyVolume, func(i, j int) bool {
				return ed.PartyVolume[i].Party < ed.PartyVolume[j].Party
			})
		}
		epochData = append(epochData, ed)
	}

	stats := make([]*snapshotpb.VolumeDiscountStats, 0, len(e.factorsByParty))
	for partyID, discountStats := range e.factorsByParty {
		stats = append(stats, &snapshotpb.VolumeDiscountStats{
			Party:           partyID.String(),
			DiscountFactors: discountStats.DiscountFactors.IntoDiscountFactorsProto(),
		})
	}
	slices.SortStableFunc(stats, func(a, b *snapshotpb.VolumeDiscountStats) int {
		return strings.Compare(a.Party, b.Party)
	})

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_VolumeDiscountProgram{
			VolumeDiscountProgram: &snapshotpb.VolumeDiscountProgram{
				CurrentProgram:     e.getProgram(e.currentProgram),
				NewProgram:         e.getProgram(e.newProgram),
				Parties:            parties,
				EpochDataIndex:     uint64(e.epochDataIndex),
				AveragePartyVolume: avgPartyVolumes,
				EpochPartyVolumes:  epochData,
				FactorsByParty:     stats,
				LastProgramVersion: e.latestProgramVersion,
				ProgramHasEnded:    e.programHasEnded,
			},
		},
	}
	return proto.Marshal(payload)
}

func (e *SnapshottedEngine) getProgram(program *types.VolumeDiscountProgram) *vegapb.VolumeDiscountProgram {
	if program == nil {
		return nil
	}
	return program.IntoProto()
}

func NewSnapshottedEngine(broker Broker, mat MarketActivityTracker) *SnapshottedEngine {
	se := &SnapshottedEngine{
		Engine: New(broker, mat),
		pl:     types.Payload{},
	}

	return se
}
