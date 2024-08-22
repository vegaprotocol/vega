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
	_VolumeRebateProgram  struct{}
	VolumeRebateProgramID = ID[_VolumeRebateProgram]

	VolumeRebateProgram struct {
		ID                    VolumeRebateProgramID
		Version               uint64
		BenefitTiers          []*vega.VolumeRebateBenefitTier
		EndOfProgramTimestamp time.Time
		WindowLength          uint64
		VegaTime              time.Time
		EndedAt               *time.Time
		SeqNum                uint64
	}
)

func VolumeRebateProgramFromProto(proto *vega.VolumeRebateProgram, vegaTime time.Time, seqNum uint64) *VolumeRebateProgram {
	// set the tier numbers accordingly
	for i := range proto.BenefitTiers {
		proto.BenefitTiers[i].TierNumber = ptr.From(uint64(i + 1))
	}
	return &VolumeRebateProgram{
		ID:                    VolumeRebateProgramID(proto.Id),
		Version:               proto.Version,
		BenefitTiers:          proto.BenefitTiers,
		EndOfProgramTimestamp: time.Unix(proto.EndOfProgramTimestamp, 0),
		WindowLength:          proto.WindowLength,
		VegaTime:              vegaTime,
		SeqNum:                seqNum,
	}
}

func (rp VolumeRebateProgram) ToProto() *v2.VolumeRebateProgram {
	var endedAt *int64
	if rp.EndedAt != nil {
		endedAt = ptr.From(rp.EndedAt.UnixNano())
	}

	// While the original program proto from core sends EndOfProgramTimestamp as
	// a timestamp in unix seconds, for the data node API, we publish it as a
	// unix timestamp in nanoseconds as the GraphQL API timestamp will incorrectly
	// treat the timestamp as nanos.
	return &v2.VolumeRebateProgram{
		Id:                    rp.ID.String(),
		Version:               rp.Version,
		BenefitTiers:          rp.BenefitTiers,
		EndOfProgramTimestamp: rp.EndOfProgramTimestamp.UnixNano(),
		WindowLength:          rp.WindowLength,
		EndedAt:               endedAt,
	}
}
