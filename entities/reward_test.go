package entities_test

import (
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewardFromProto(t *testing.T) {
	now := time.Now()

	pbReward := eventspb.RewardPayoutEvent{
		Party:                "a0b1",
		Asset:                "c2d3",
		EpochSeq:             "42",
		Amount:               "123456789",
		PercentOfTotalReward: "3.14",
		Timestamp:            now.UnixNano(),
	}

	reward, err := entities.RewardFromProto(pbReward)
	require.NoError(t, err)
	assert.Equal(t, "a0b1", reward.PartyHexID())
	assert.Equal(t, "c2d3", reward.AssetHexID())
	assert.Equal(t, int64(42), reward.EpochID)
	assert.InDelta(t, 3.14, reward.PercentOfTotal, 0.001)
	fmt.Printf("%v - %v\n", now, reward.VegaTime)
	assert.True(t, now.Equal(reward.VegaTime))

}
