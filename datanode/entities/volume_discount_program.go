package entities

import (
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type (
	_VolumeDiscountProgram  struct{}
	VolumeDiscountProgramID = ID[_VolumeDiscountProgram]

	VolumeDiscountProgram struct {
		ID                    VolumeDiscountProgramID
		Version               uint64
		BenefitTiers          []*vega.VolumeBenefitTier
		EndOfProgramTimestamp time.Time
		WindowLength          uint64
		VegaTime              time.Time
		EndedAt               *time.Time
	}
)

func VolumeDiscountProgramFromProto(proto *vega.VolumeDiscountProgram, vegaTime time.Time) *VolumeDiscountProgram {
	return &VolumeDiscountProgram{
		ID:                    VolumeDiscountProgramID(proto.Id),
		Version:               proto.Version,
		BenefitTiers:          proto.BenefitTiers,
		EndOfProgramTimestamp: time.Unix(proto.EndOfProgramTimestamp, 0),
		WindowLength:          proto.WindowLength,
		VegaTime:              vegaTime,
	}
}

func (rp VolumeDiscountProgram) ToProto() *v2.VolumeDiscountProgram {
	var endedAt *int64
	if rp.EndedAt != nil {
		endedAt = ptr.From(rp.EndedAt.UnixNano())
	}

	return &v2.VolumeDiscountProgram{
		Id:                    rp.ID.String(),
		Version:               rp.Version,
		BenefitTiers:          rp.BenefitTiers,
		EndOfProgramTimestamp: rp.EndOfProgramTimestamp.Unix(),
		WindowLength:          rp.WindowLength,
		EndedAt:               endedAt,
	}
}
