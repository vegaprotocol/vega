package entities

import (
	"time"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	VolumeDiscountStatsUpdated struct {
		AtEpoch                  uint64
		PartyVolumeDiscountStats []*eventspb.PartyVolumeDiscountStats
		VegaTime                 time.Time
	}

	VolumeDiscountStatsCursor struct {
		VegaTime time.Time
		AtEpoch  uint64
	}
)

func NewVolumeDiscountStatsFromProto(vestingStatsProto *eventspb.VolumeDiscountStatsUpdated, vegaTime time.Time) (*VolumeDiscountStatsUpdated, error) {
	return &VolumeDiscountStatsUpdated{
		AtEpoch:                  vestingStatsProto.AtEpoch,
		PartyVolumeDiscountStats: vestingStatsProto.Stats,
		VegaTime:                 vegaTime,
	}, nil
}
