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

package entities

import (
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type (
	_ReferralProgram  struct{}
	ReferralProgramID = ID[_ReferralProgram]

	ReferralProgram struct {
		ID                    ReferralProgramID
		Version               uint64
		BenefitTiers          []*vega.BenefitTier
		EndOfProgramTimestamp time.Time
		WindowLength          uint64
		StakingTiers          []*vega.StakingTier
		VegaTime              time.Time
		EndedAt               *time.Time
		SeqNum                uint64
	}
)

func ReferralProgramFromProto(proto *vega.ReferralProgram, vegaTime time.Time, seqNum uint64) *ReferralProgram {
	for i := range proto.BenefitTiers {
		proto.BenefitTiers[i].TierNumber = ptr.From(uint64(i + 1))
	}
	return &ReferralProgram{
		ID:                    ReferralProgramID(proto.Id),
		Version:               proto.Version,
		BenefitTiers:          proto.BenefitTiers,
		EndOfProgramTimestamp: time.Unix(proto.EndOfProgramTimestamp, 0),
		WindowLength:          proto.WindowLength,
		StakingTiers:          proto.StakingTiers,
		VegaTime:              vegaTime,
		SeqNum:                seqNum,
	}
}

func (rp ReferralProgram) ToProto() *v2.ReferralProgram {
	var endedAt *int64
	if rp.EndedAt != nil {
		endedAt = ptr.From(rp.EndedAt.UnixNano())
	}

	// While the original referral program proto from core sends EndOfProgramTimestamp as a timestamp in unix seconds,
	// For the data node API, we publish it as a unix timestamp in nanoseconds as the GraphQL API timestamp will incorrectly
	// treat the timestamp as nanos.
	return &v2.ReferralProgram{
		Id:                    rp.ID.String(),
		Version:               rp.Version,
		BenefitTiers:          rp.BenefitTiers,
		EndOfProgramTimestamp: rp.EndOfProgramTimestamp.UnixNano(),
		WindowLength:          rp.WindowLength,
		StakingTiers:          rp.StakingTiers,
		EndedAt:               endedAt,
	}
}
