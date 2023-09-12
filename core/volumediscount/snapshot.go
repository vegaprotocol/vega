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

package volumediscount

import (
	"context"
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var (
	key                        = (&types.PayloadVolumeDiscountProgram{}).Key()
	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for volume discount program snapshot")
	hashKeys                   = []string{key}
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

	if e.latestProgramVersion < e.currentProgram.Version {
		e.latestProgramVersion = e.currentProgram.Version
	}
	e.programHasEnded = false
}

func (e *Engine) loadNewProgramFromSnapshot(program *vegapb.VolumeDiscountProgram) {
	if program == nil {
		e.newProgram = nil
		return
	}

	e.newProgram = types.NewVolumeDiscountProgramFromProto(program)

	if e.latestProgramVersion < e.newProgram.Version {
		e.latestProgramVersion = e.newProgram.Version
	}
}

func (e *SnapshottedEngine) restore(vdp *snapshotpb.VolumeDiscountProgram) error {
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

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_VolumeDiscountProgram{
			VolumeDiscountProgram: &snapshotpb.VolumeDiscountProgram{
				CurrentProgram:     e.getProgram(e.currentProgram),
				NewProgram:         e.getProgram(e.newProgram),
				Parties:            parties,
				EpochDataIndex:     uint64(e.epochDataIndex),
				AveragePartyVolume: avgPartyVolumes,
				EpochPartyVolumes:  epochData,
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
