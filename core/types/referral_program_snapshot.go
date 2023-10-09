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
