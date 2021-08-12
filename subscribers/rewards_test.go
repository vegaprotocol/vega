package subscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"

	"github.com/stretchr/testify/assert"
)

func TestFirstRewardMessage(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	// Create a reward event and push it to the subscriber
	// re.Push(rpe)

	// Now query for the reward details for that party
	details, err := re.GetRewardDetails(ctx, partyID)

	assert.NoError(t, err)

	assert.Equal(t, "BTC", details.AssetID)
	assert.Equal(t, 100, details.LastReward)
	assert.Equal(t, 100, details.TotalReward)
	assert.Equal(t, 10.0, details.LastPercentageAmount)
}

func TestPartyWithNoReward(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	details, err := re.GetRewardDetails(ctx, partyID)

	assert.Error(t, err)
	assert.Nil(t, details)
}
