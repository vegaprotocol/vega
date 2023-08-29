package entities

import (
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type (
	_ReferralProgram struct{}
	ReferralID       = ID[_ReferralProgram]

	ReferralProgram struct {
		ID                    ReferralID
		Version               uint64
		BenefitTiers          []*vega.BenefitTier
		EndOfProgramTimestamp time.Time
		WindowLength          uint64
		VegaTime              time.Time
		EndedAt               *time.Time
	}
)

func ReferralProgramFromProto(proto *vega.ReferralProgram, vegaTime time.Time) *ReferralProgram {
	return &ReferralProgram{
		ID:                    ReferralID(proto.Id),
		Version:               proto.Version,
		EndOfProgramTimestamp: time.Unix(proto.EndOfProgramTimestamp, 0),
		WindowLength:          proto.WindowLength,
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
		EndedAt:               endedAt,
	}
}
