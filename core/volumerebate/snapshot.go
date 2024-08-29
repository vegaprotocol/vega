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

package volumerebate

import (
	"context"
	"fmt"
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
	key      = (&types.PayloadVolumeRebateProgram{}).Key()
	hashKeys = []string{key}
)

type SnapshottedEngine struct {
	*Engine

	pl types.Payload
}

func (e *SnapshottedEngine) Namespace() types.SnapshotNamespace {
	return types.VolumeRebateProgramSnapshot
}

func (e *SnapshottedEngine) Keys() []string {
	return hashKeys
}

func (e *SnapshottedEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *Engine) loadCurrentProgramFromSnapshot(program *vegapb.VolumeRebateProgram) {
	if program == nil {
		e.currentProgram = nil
		return
	}

	e.currentProgram = types.NewVolumeRebateProgramFromProto(program)
}

func (e *Engine) loadNewProgramFromSnapshot(program *vegapb.VolumeRebateProgram) {
	if program == nil {
		e.newProgram = nil
		return
	}

	e.newProgram = types.NewVolumeRebateProgramFromProto(program)
}

func (e *SnapshottedEngine) restore(vdp *snapshotpb.VolumeRebateProgram) error {
	e.latestProgramVersion = vdp.LastProgramVersion
	e.programHasEnded = vdp.ProgramHasEnded
	e.loadCurrentProgramFromSnapshot(vdp.CurrentProgram)
	e.loadNewProgramFromSnapshot(vdp.NewProgram)
	for _, v := range vdp.Parties {
		e.parties[v] = struct{}{}
	}
	for _, pv := range vdp.PartyRebateData {
		fraction, err := num.DecimalFromString(pv.Fraction)
		if err != nil {
			return err
		}
		e.fractionPerParty[pv.Party] = fraction

		makerFee, overflow := num.UintFromString(pv.MakerFeeReceived, 10)
		if overflow {
			return err
		}
		e.makerFeesReceivedInWindowPerParty[pv.Party] = makerFee
	}

	for _, stats := range vdp.FactorsByParty {
		factor, err := num.DecimalFromString(stats.RebateFactor)
		if err != nil {
			return fmt.Errorf("could not parse string %q into decimal: %w", stats.RebateFactor, err)
		}
		e.factorsByParty[types.PartyID(stats.Party)] = types.VolumeRebateStats{
			RebateFactor: factor,
		}
	}

	return nil
}

func (e *SnapshottedEngine) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch data := p.Data.(type) {
	case *types.PayloadVolumeRebateProgram:
		return nil, e.restore(data.VolumeRebateProgram)
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
		return e.serialiseRebateVolumeProgram()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *SnapshottedEngine) serialiseRebateVolumeProgram() ([]byte, error) {
	parties := make([]string, 0, len(e.parties))
	for pi := range e.parties {
		parties = append(parties, pi)
	}
	sort.Strings(parties)

	partyData := make([]*snapshotpb.PartyRebateData, 0, len(e.fractionPerParty))
	for pi, d := range e.fractionPerParty {
		partyData = append(partyData, &snapshotpb.PartyRebateData{
			Party:            pi,
			Fraction:         d.String(),
			MakerFeeReceived: e.makerFeesReceivedInWindowPerParty[pi].String(),
		})
	}
	sort.Slice(partyData, func(i, j int) bool {
		return partyData[i].Party < partyData[j].Party
	})

	stats := make([]*snapshotpb.VolumeRebateStats, 0, len(e.factorsByParty))
	for partyID, rebateStats := range e.factorsByParty {
		stats = append(stats, &snapshotpb.VolumeRebateStats{
			Party:        partyID.String(),
			RebateFactor: rebateStats.RebateFactor.String(),
		})
	}
	slices.SortStableFunc(stats, func(a, b *snapshotpb.VolumeRebateStats) int {
		return strings.Compare(a.Party, b.Party)
	})

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_VolumeRebateProgram{
			VolumeRebateProgram: &snapshotpb.VolumeRebateProgram{
				CurrentProgram:     e.getProgram(e.currentProgram),
				NewProgram:         e.getProgram(e.newProgram),
				Parties:            parties,
				PartyRebateData:    partyData,
				FactorsByParty:     stats,
				LastProgramVersion: e.latestProgramVersion,
				ProgramHasEnded:    e.programHasEnded,
			},
		},
	}
	return proto.Marshal(payload)
}

func (e *SnapshottedEngine) getProgram(program *types.VolumeRebateProgram) *vegapb.VolumeRebateProgram {
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
