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
		SeqNum                uint64
	}
)

func VolumeDiscountProgramFromProto(proto *vega.VolumeDiscountProgram, vegaTime time.Time, seqNum uint64) *VolumeDiscountProgram {
	return &VolumeDiscountProgram{
		ID:                    VolumeDiscountProgramID(proto.Id),
		Version:               proto.Version,
		BenefitTiers:          proto.BenefitTiers,
		EndOfProgramTimestamp: time.Unix(proto.EndOfProgramTimestamp, 0),
		WindowLength:          proto.WindowLength,
		VegaTime:              vegaTime,
		SeqNum:                seqNum,
	}
}

func (rp VolumeDiscountProgram) ToProto() *v2.VolumeDiscountProgram {
	var endedAt *int64
	if rp.EndedAt != nil {
		endedAt = ptr.From(rp.EndedAt.UnixNano())
	}

	// While the original program proto from core sends EndOfProgramTimestamp as
	// a timestamp in unix seconds, for the data node API, we publish it as a
	// unix timestamp in nanoseconds as the GraphQL API timestamp will incorrectly
	// treat the timestamp as nanos.
	return &v2.VolumeDiscountProgram{
		Id:                    rp.ID.String(),
		Version:               rp.Version,
		BenefitTiers:          rp.BenefitTiers,
		EndOfProgramTimestamp: rp.EndOfProgramTimestamp.UnixNano(),
		WindowLength:          rp.WindowLength,
		EndedAt:               endedAt,
	}
}
