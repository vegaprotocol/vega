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
	}
)

func ReferralProgramFromProto(proto *vega.ReferralProgram, vegaTime time.Time) *ReferralProgram {
	return &ReferralProgram{
		ID:                    ReferralProgramID(proto.Id),
		Version:               proto.Version,
		BenefitTiers:          proto.BenefitTiers,
		EndOfProgramTimestamp: time.Unix(proto.EndOfProgramTimestamp, 0),
		WindowLength:          proto.WindowLength,
		StakingTiers:          proto.StakingTiers,
		VegaTime:              vegaTime,
	}
}

func (rp ReferralProgram) ToProto() *v2.ReferralProgram {
	var endedAt *int64
	if rp.EndedAt != nil {
		endedAt = ptr.From(rp.EndedAt.UnixNano())
	}

	return &v2.ReferralProgram{
		Id:                    rp.ID.String(),
		Version:               rp.Version,
		BenefitTiers:          rp.BenefitTiers,
		EndOfProgramTimestamp: rp.EndOfProgramTimestamp.Unix(),
		WindowLength:          rp.WindowLength,
		StakingTiers:          rp.StakingTiers,
		EndedAt:               endedAt,
	}
}
