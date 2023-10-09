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

package types

import (
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type PayloadReferralProgramState struct {
	FactorByReferee    []*snapshotpb.FactorByReferee
	CurrentProgram     *vegapb.ReferralProgram
	NewProgram         *vegapb.ReferralProgram
	LastProgramVersion uint64
	ProgramHasEnded    bool
	Sets               []*snapshotpb.ReferralSet
}

func (p *PayloadReferralProgramState) Key() string {
	return "referral"
}

func (*PayloadReferralProgramState) Namespace() SnapshotNamespace {
	return ReferralProgramSnapshot
}

func (p *PayloadReferralProgramState) IntoProto() *snapshotpb.Payload_ReferralProgram {
	return &snapshotpb.Payload_ReferralProgram{
		ReferralProgram: &snapshotpb.ReferralProgramData{
			FactorByReferee:    p.FactorByReferee,
			CurrentProgram:     p.CurrentProgram,
			NewProgram:         p.NewProgram,
			LastProgramVersion: p.LastProgramVersion,
			ProgramHasEnded:    p.ProgramHasEnded,
		},
	}
}

func (*PayloadReferralProgramState) isPayload() {}

func (p *PayloadReferralProgramState) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadReferralProgramStateFromProto(payload *snapshotpb.Payload_ReferralProgram) *PayloadReferralProgramState {
	return &PayloadReferralProgramState{
		FactorByReferee:    payload.ReferralProgram.FactorByReferee,
		CurrentProgram:     payload.ReferralProgram.CurrentProgram,
		NewProgram:         payload.ReferralProgram.NewProgram,
		LastProgramVersion: payload.ReferralProgram.LastProgramVersion,
		ProgramHasEnded:    payload.ReferralProgram.ProgramHasEnded,
	}
}
