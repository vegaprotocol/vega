package sqlstore

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
)

type (
	VolumeDiscountStats struct {
		*ConnectionSource
	}
)

func NewVolumeDiscountStats(connectionSource *ConnectionSource) *VolumeDiscountStats {
	return &VolumeDiscountStats{
		ConnectionSource: connectionSource,
	}
}

func (t *VolumeDiscountStats) Add(context.Context, *entities.VolumeDiscountStatsUpdated) error {
	// TODO Implement the API.
	return nil
}
