package sqlstore

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
)

type (
	VestingStats struct {
		*ConnectionSource
	}
)

func NewVestingStats(connectionSource *ConnectionSource) *VestingStats {
	return &VestingStats{
		ConnectionSource: connectionSource,
	}
}

func (t *VestingStats) Add(context.Context, *entities.VestingStatsUpdated) error {
	// TODO Implement the API.
	return nil
}
