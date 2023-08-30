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

type PayloadCurrentReferralProgram struct {
	CurrentReferralProgram *vegapb.ReferralProgram
}

func (p *PayloadCurrentReferralProgram) Key() string {
	return "currentReferralProgram"
}

func (*PayloadCurrentReferralProgram) Namespace() SnapshotNamespace {
	return ReferralProgramSnapshot
}

func (p *PayloadCurrentReferralProgram) IntoProto() *snapshotpb.Payload_CurrentReferralProgram {
	return &snapshotpb.Payload_CurrentReferralProgram{
		CurrentReferralProgram: &snapshotpb.CurrentReferralProgram{
			ReferralProgram: p.CurrentReferralProgram,
		},
	}
}

func (*PayloadCurrentReferralProgram) isPayload() {}

func (p *PayloadCurrentReferralProgram) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadCurrentReferralProgramFromProto(payload *snapshotpb.Payload_CurrentReferralProgram) *PayloadCurrentReferralProgram {
	return &PayloadCurrentReferralProgram{
		CurrentReferralProgram: payload.CurrentReferralProgram.GetReferralProgram(),
	}
}

type PayloadNewReferralProgram struct {
	NewReferralProgram *vegapb.ReferralProgram
}

func (p *PayloadNewReferralProgram) Key() string {
	return "newReferralProgram"
}

func (*PayloadNewReferralProgram) Namespace() SnapshotNamespace {
	return ReferralProgramSnapshot
}

func (p *PayloadNewReferralProgram) IntoProto() *snapshotpb.Payload_NewReferralProgram {
	return &snapshotpb.Payload_NewReferralProgram{
		NewReferralProgram: &snapshotpb.NewReferralProgram{
			ReferralProgram: p.NewReferralProgram,
		},
	}
}

func (*PayloadNewReferralProgram) isPayload() {}

func (p *PayloadNewReferralProgram) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadNewReferralProgramFromProto(teamsPayload *snapshotpb.Payload_NewReferralProgram) *PayloadNewReferralProgram {
	return &PayloadNewReferralProgram{
		NewReferralProgram: teamsPayload.NewReferralProgram.GetReferralProgram(),
	}
}

type PayloadReferralSets struct {
	Sets *snapshotpb.ReferralSets
}

func (p *PayloadReferralSets) Key() string {
	return "referralSets"
}

func (*PayloadReferralSets) Namespace() SnapshotNamespace {
	return ReferralProgramSnapshot
}

func (p *PayloadReferralSets) IntoProto() *snapshotpb.Payload_ReferralSets {
	return &snapshotpb.Payload_ReferralSets{
		ReferralSets: p.Sets,
	}
}

func (*PayloadReferralSets) isPayload() {}

func (p *PayloadReferralSets) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadReferralSetsFromProto(p *snapshotpb.Payload_ReferralSets) *PayloadReferralSets {
	return &PayloadReferralSets{
		Sets: p.ReferralSets,
	}
}
