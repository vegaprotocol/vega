package subscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"
	v1 "code.vegaprotocol.io/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"
)

func TestFirstRewardMessage(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	rpe1 := v1.RewardPayoutEvent{
		Party:                partyID,
		FromAccount:          "From",
		ToAccount:            "To",
		EpochSeq:             "1",
		Asset:                "BTC",
		Amount:               100,
		PercentOfTotalReward: "0.1",
	}
	re.UpdateRewards(rpe1)

	// Now query for the reward details for that party
	details, err := re.GetRewardDetails(ctx, partyID)

	assert.NoError(t, err)

	assert.Equal(t, "BTC", details.AssetID)
	assert.EqualValues(t, 100, details.LastReward)
	assert.EqualValues(t, 100, details.TotalReward)
	assert.Equal(t, 0.1, details.LastPercentageAmount)
}

func TestTwoUpdates(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	// Create a reward event and push it to the subscriber
	rpe1 := v1.RewardPayoutEvent{
		Party:                partyID,
		FromAccount:          "From",
		ToAccount:            "To",
		EpochSeq:             "1",
		Asset:                "BTC",
		Amount:               100,
		PercentOfTotalReward: "0.1",
	}
	re.UpdateRewards(rpe1)

	rpe2 := v1.RewardPayoutEvent{
		Party:                partyID,
		FromAccount:          "From",
		ToAccount:            "To",
		EpochSeq:             "2",
		Asset:                "BTC",
		Amount:               50,
		PercentOfTotalReward: "0.2",
	}
	re.UpdateRewards(rpe2)

	// Now query for the reward details for that party
	details, err := re.GetRewardDetails(ctx, partyID)
	assert.NoError(t, err)
	assert.Equal(t, "BTC", details.AssetID)
	assert.EqualValues(t, 50, details.LastReward)
	assert.EqualValues(t, 150, details.TotalReward)
	assert.Equal(t, 0.2, details.LastPercentageAmount)
}

func TestPartyWithNoReward(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	details, err := re.GetRewardDetails(ctx, partyID)

	assert.Error(t, err)
	assert.Nil(t, details)
}
