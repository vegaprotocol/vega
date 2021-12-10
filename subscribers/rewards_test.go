package subscribers_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

func TestFirstRewardMessage(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	now := time.Now().UnixNano()
	evt := events.NewRewardPayout(ctx, now, partyID, "1", "BTC", num.NewUint(100), 0.1)
	re.Push(evt)

	// Now query for the reward details for that party
	details, err := re.GetRewardDetails(ctx, partyID)

	assert.NoError(t, err)
	assert.NotNil(t, details)

	assert.Equal(t, 1, len(details.RewardDetails))
	assert.Equal(t, "BTC", details.RewardDetails[0].Asset)
	assert.Equal(t, "100", details.RewardDetails[0].TotalForAsset)

	assert.Equal(t, 1, len(details.RewardDetails[0].Details))
	assert.Equal(t, "100", details.RewardDetails[0].Details[0].Amount)
	assert.Equal(t, "BTC", details.RewardDetails[0].Details[0].AssetId)
	assert.EqualValues(t, 1, details.RewardDetails[0].Details[0].Epoch)
	assert.Equal(t, "party1", details.RewardDetails[0].Details[0].PartyId)
	assert.Equal(t, "0.10000", details.RewardDetails[0].Details[0].PercentageOfTotal)
	assert.EqualValues(t, now, details.RewardDetails[0].Details[0].ReceivedAt)
}

func TestTwoRewardsSamePartyAndAsset(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	// Create a reward event and push it to the subscriber
	now := time.Now().UnixNano()
	evt := events.NewRewardPayout(ctx, now, partyID, "1", "BTC", num.NewUint(100), 0.1)
	re.Push(evt)
	evt2 := events.NewRewardPayout(ctx, now, partyID, "2", "BTC", num.NewUint(50), 0.2)
	re.Push(evt2)

	// Now query for the reward details for that party
	details, err := re.GetRewardDetails(ctx, partyID)

	sort.Slice(details.RewardDetails[0].Details, func(i, j int) bool {
		return details.RewardDetails[0].Details[i].PercentageOfTotal < details.RewardDetails[0].Details[j].PercentageOfTotal
	})

	assert.NotNil(t, details)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(details.RewardDetails))
	assert.Equal(t, "BTC", details.RewardDetails[0].Asset)
	assert.Equal(t, "150", details.RewardDetails[0].TotalForAsset)

	assert.Equal(t, 2, len(details.RewardDetails[0].Details))
	assert.Equal(t, "100", details.RewardDetails[0].Details[0].Amount)
	assert.Equal(t, "BTC", details.RewardDetails[0].Details[0].AssetId)
	assert.EqualValues(t, 1, details.RewardDetails[0].Details[0].Epoch)
	assert.Equal(t, "party1", details.RewardDetails[0].Details[0].PartyId)
	assert.Equal(t, "0.10000", details.RewardDetails[0].Details[0].PercentageOfTotal)
	assert.EqualValues(t, now, details.RewardDetails[0].Details[0].ReceivedAt)

	assert.Equal(t, "50", details.RewardDetails[0].Details[1].Amount)
	assert.Equal(t, "BTC", details.RewardDetails[0].Details[1].AssetId)
	assert.EqualValues(t, 2, details.RewardDetails[0].Details[1].Epoch)
	assert.Equal(t, "party1", details.RewardDetails[0].Details[1].PartyId)
	assert.Equal(t, "0.20000", details.RewardDetails[0].Details[1].PercentageOfTotal)
	assert.EqualValues(t, now, details.RewardDetails[0].Details[1].ReceivedAt)
}

func TestTwoDifferentAssetsSameParty(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	// Create a reward event and push it to the subscriber
	now := time.Now().UnixNano()
	evt := events.NewRewardPayout(ctx, now, partyID, "1", "BTC", num.NewUint(100), 0.1)
	re.Push(evt)
	evt2 := events.NewRewardPayout(ctx, now, partyID, "2", "ETH", num.NewUint(50), 0.2)
	re.Push(evt2)

	// Now query for the reward details for that party
	details, err := re.GetRewardDetails(ctx, partyID)

	sort.Slice(details.RewardDetails[0].Details, func(i, j int) bool {
		return details.RewardDetails[0].Details[i].PercentageOfTotal < details.RewardDetails[0].Details[j].PercentageOfTotal
	})

	// first sort details
	sort.Slice(details.RewardDetails, func(i, j int) bool { return details.RewardDetails[i].Asset < details.RewardDetails[j].Asset })

	for _, det := range details.RewardDetails {
		sort.Slice(det.Details, func(i, j int) bool { return det.Details[i].PercentageOfTotal < det.Details[j].PercentageOfTotal })
	}

	assert.NotNil(t, details)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(details.RewardDetails))
	assert.Equal(t, "BTC", details.RewardDetails[0].Asset)
	assert.Equal(t, "100", details.RewardDetails[0].TotalForAsset)
	assert.Equal(t, "ETH", details.RewardDetails[1].Asset)
	assert.Equal(t, "50", details.RewardDetails[1].TotalForAsset)

	assert.Equal(t, 1, len(details.RewardDetails[0].Details))
	assert.Equal(t, "100", details.RewardDetails[0].Details[0].Amount)
	assert.Equal(t, "BTC", details.RewardDetails[0].Details[0].AssetId)
	assert.EqualValues(t, 1, details.RewardDetails[0].Details[0].Epoch)
	assert.Equal(t, "party1", details.RewardDetails[0].Details[0].PartyId)
	assert.Equal(t, "0.10000", details.RewardDetails[0].Details[0].PercentageOfTotal)
	assert.EqualValues(t, now, details.RewardDetails[0].Details[0].ReceivedAt)

	assert.Equal(t, 1, len(details.RewardDetails[1].Details))
	assert.Equal(t, "50", details.RewardDetails[1].Details[0].Amount)
	assert.Equal(t, "ETH", details.RewardDetails[1].Details[0].AssetId)
	assert.EqualValues(t, 2, details.RewardDetails[1].Details[0].Epoch)
	assert.Equal(t, "party1", details.RewardDetails[1].Details[0].PartyId)
	assert.Equal(t, "0.20000", details.RewardDetails[1].Details[0].PercentageOfTotal)
	assert.EqualValues(t, now, details.RewardDetails[1].Details[0].ReceivedAt)
}

func TestPartyWithNoRewards(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	details, err := re.GetRewardDetails(ctx, partyID)

	assert.NoError(t, err)
	assert.NotNil(t, details)
	assert.Zero(t, len(details.RewardDetails))
}
