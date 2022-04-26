package subscribers_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

func TestFirstRewardMessage(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	now := time.Now().UnixNano()
	evt := events.NewRewardPayout(ctx, now, partyID, "1", "BTC", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, "123")
	re.Push(evt)

	// Check the summary
	summary := re.GetRewardSummaries(ctx, partyID, nil)
	assert.Equal(t, "100", summary[0].Amount)
	assert.Equal(t, "BTC", summary[0].AssetId)

	// Now query for the reward rewards for that party
	rewards := re.GetRewards(ctx, partyID, 0, 10, true)

	assert.Equal(t, 1, len(rewards))
	assert.Equal(t, "100", rewards[0].Amount)
	assert.Equal(t, "BTC", rewards[0].AssetId)
	assert.EqualValues(t, 1, rewards[0].Epoch)
	assert.Equal(t, "party1", rewards[0].PartyId)
	assert.Equal(t, "0.10000", rewards[0].PercentageOfTotal)
	assert.EqualValues(t, now, rewards[0].ReceivedAt)
}

func TestTwoRewardsSamePartyAndAsset(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	// Create a reward event and push it to the subscriber
	now := time.Now().UnixNano()
	evt := events.NewRewardPayout(ctx, now, partyID, "1", "BTC", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, "123")
	re.Push(evt)
	evt2 := events.NewRewardPayout(ctx, now, partyID, "2", "BTC", num.NewUint(50), num.DecimalFromFloat(0.2), types.AccountTypeMakerFeeReward, "123")
	re.Push(evt2)

	// Now query for the reward summaries for that party
	summaries := re.GetRewardSummaries(ctx, partyID, nil)
	assert.Equal(t, 1, len(summaries))
	assert.Equal(t, "BTC", summaries[0].AssetId)
	assert.Equal(t, "150", summaries[0].Amount)

	// Now query each individual reward for that party
	rewards := re.GetRewards(ctx, partyID, 0, 10, true)

	sort.Slice(rewards, func(i, j int) bool {
		return rewards[i].PercentageOfTotal < rewards[j].PercentageOfTotal
	})

	assert.Equal(t, 2, len(rewards))
	assert.Equal(t, "100", rewards[0].Amount)
	assert.Equal(t, "BTC", rewards[0].AssetId)
	assert.EqualValues(t, 1, rewards[0].Epoch)
	assert.Equal(t, "party1", rewards[0].PartyId)
	assert.Equal(t, "0.10000", rewards[0].PercentageOfTotal)
	assert.EqualValues(t, now, rewards[0].ReceivedAt)

	assert.Equal(t, "50", rewards[1].Amount)
	assert.Equal(t, "BTC", rewards[1].AssetId)
	assert.EqualValues(t, 2, rewards[1].Epoch)
	assert.Equal(t, "party1", rewards[1].PartyId)
	assert.Equal(t, "0.20000", rewards[1].PercentageOfTotal)
	assert.EqualValues(t, now, rewards[1].ReceivedAt)
}

func TestTwoDifferentAssetsSameParty(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	// Create a reward event and push it to the subscriber
	now := time.Now().UnixNano()
	evt := events.NewRewardPayout(ctx, now, partyID, "1", "BTC", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, "123")
	re.Push(evt)
	evt2 := events.NewRewardPayout(ctx, now, partyID, "2", "ETH", num.NewUint(50), num.DecimalFromFloat(0.2), types.AccountTypeMakerFeeReward, "123")
	re.Push(evt2)

	// Now query for the reward summaries for that party
	summaries := re.GetRewardSummaries(ctx, partyID, nil)

	// first sort details
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].AssetId < summaries[j].AssetId })

	assert.NotNil(t, summaries)
	assert.Equal(t, 2, len(summaries))
	assert.Equal(t, "BTC", summaries[0].AssetId)
	assert.Equal(t, "100", summaries[0].Amount)
	assert.Equal(t, "ETH", summaries[1].AssetId)
	assert.Equal(t, "50", summaries[1].Amount)

	// Now query for the individual rewards for that party
	rewards := re.GetRewards(ctx, partyID, 0, 10, true)

	sort.Slice(rewards, func(i, j int) bool {
		return rewards[i].PercentageOfTotal < rewards[j].PercentageOfTotal
	})

	assert.Equal(t, 2, len(rewards))

	assert.Equal(t, "100", rewards[0].Amount)
	assert.Equal(t, "BTC", rewards[0].AssetId)
	assert.EqualValues(t, 1, rewards[0].Epoch)
	assert.Equal(t, "party1", rewards[0].PartyId)
	assert.Equal(t, "0.10000", rewards[0].PercentageOfTotal)
	assert.EqualValues(t, now, rewards[0].ReceivedAt)

	assert.Equal(t, "50", rewards[1].Amount)
	assert.Equal(t, "ETH", rewards[1].AssetId)
	assert.EqualValues(t, 2, rewards[1].Epoch)
	assert.Equal(t, "party1", rewards[1].PartyId)
	assert.Equal(t, "0.20000", rewards[1].PercentageOfTotal)
	assert.EqualValues(t, now, rewards[1].ReceivedAt)
}

func TestPartyWithNoRewards(t *testing.T) {
	ctx := context.Background()
	partyID := "party1"
	re := subscribers.NewRewards(ctx, logging.NewTestLogger(), true)

	details := re.GetRewards(ctx, partyID, 0, 10, true)

	assert.Zero(t, len(details))
}

type testCase struct {
	skip        uint64
	limit       uint64
	descending  bool
	expected    []*vega.Reward
	description string
}

type rewards []*vega.Reward

func TestPaginateRewards(t *testing.T) {
	r1 := &vega.Reward{Epoch: 1}
	r2 := &vega.Reward{Epoch: 2}
	r3 := &vega.Reward{Epoch: 3}
	testRewards := rewards{r1, r2, r3}

	tc1 := testCase{0, 2, false, rewards{r1, r2}, "First Two"}
	tc2 := testCase{1, 2, false, rewards{r2, r3}, "Skip one, take two"}
	tc3 := testCase{4, 2, false, rewards{}, "Skip past end"}
	tc4 := testCase{0, 4, false, rewards{r1, r2, r3}, "First > length"}
	tc5 := testCase{3, 0, false, rewards{}, "Skip everything"}
	tc6 := testCase{0, 2, true, rewards{r3, r2}, "Last Two"}
	tc7 := testCase{1, 1, true, rewards{r2}, "Last but one"}
	tc8 := testCase{4, 1, true, rewards{}, "Skip before beginning"}
	tc9 := testCase{1, 4, true, rewards{r2, r1}, "Last before beginning"}

	cases := []testCase{tc1, tc2, tc3, tc4, tc5, tc6, tc7, tc8, tc9}
	for _, tc := range cases {
		actual := subscribers.PaginateRewards(testRewards, tc.skip, tc.limit, tc.descending)
		assert.Equal(t, tc.expected, actual, tc.description)
	}
}
