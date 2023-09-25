package entities

import (
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	VolumeDiscountStats struct {
		AtEpoch                    uint64
		PartiesVolumeDiscountStats []*eventspb.PartyVolumeDiscountStats
		VegaTime                   time.Time
	}

	FlattenVolumeDiscountStats struct {
		AtEpoch        uint64
		PartyID        string
		DiscountFactor string
		RunningVolume  string
		VegaTime       time.Time
	}

	VolumeDiscountStatsCursor struct {
		VegaTime time.Time
		AtEpoch  uint64
		PartyID  string
	}
)

func (s FlattenVolumeDiscountStats) Cursor() *Cursor {
	c := VolumeDiscountStatsCursor{
		VegaTime: s.VegaTime,
		AtEpoch:  s.AtEpoch,
		PartyID:  s.PartyID,
	}
	return NewCursor(c.ToString())
}

func (s FlattenVolumeDiscountStats) ToProtoEdge(_ ...any) (*v2.VolumeDiscountStatsEdge, error) {
	return &v2.VolumeDiscountStatsEdge{
		Node:   s.ToProto(),
		Cursor: s.Cursor().Encode(),
	}, nil
}

func (s FlattenVolumeDiscountStats) ToProto() *v2.VolumeDiscountStats {
	return &v2.VolumeDiscountStats{
		AtEpoch:        s.AtEpoch,
		PartyId:        s.PartyID,
		DiscountFactor: s.DiscountFactor,
		RunningVolume:  s.RunningVolume,
	}
}

func NewVolumeDiscountStatsFromProto(vestingStatsProto *eventspb.VolumeDiscountStatsUpdated, vegaTime time.Time) (*VolumeDiscountStats, error) {
	return &VolumeDiscountStats{
		AtEpoch:                    vestingStatsProto.AtEpoch,
		PartiesVolumeDiscountStats: vestingStatsProto.Stats,
		VegaTime:                   vegaTime,
	}, nil
}

func (c *VolumeDiscountStatsCursor) ToString() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal volume discount stats cursor: %v", err))
	}
	return string(bs)
}

func (c *VolumeDiscountStatsCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
