// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package api_test

import (
	"context"
	"io"
	"strconv"
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	"code.vegaprotocol.io/vega/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waits until the reward server has at least on subscriber
func waitForRwSubsription(ctx context.Context, ts *TestServer) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if ts.rw.GetRewardSubscribersCount() > 0 {
				return nil
			}
		}
	}
}

func TestRewardObserver(t *testing.T) {
	t.Run("rewards observer with an empty filter passes all", testObserveRewardsResponsesNoFilter)
	t.Run("rewards observer with a asset filter passes only events matching the asset", testObserveRewardsResponsesWithAssetFilter)
	t.Run("rewards observer with a party filter passes only events matching the party", testObserveRewardsResponsesWithPartyFilter)
	t.Run("rewards observer with a party/node filter passes only events matching the asset and the party", testObserveRewardsResponsesWithAssetPartyFilter)
}

func testObserveRewardsResponsesNoFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveRewardsRequest{}
	rewardEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 1, "party2", "2", "asset2", num.NewUint(200), num.DecimalFromFloat(0.2), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset2", num.NewUint(300), num.DecimalFromFloat(0.3), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset2", num.NewUint(400), num.DecimalFromFloat(0.4), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 4, "party2", "5", "asset2", num.NewUint(500), num.DecimalFromFloat(0.5), types.AccountTypeMakerFeeReward, "123"),
	}

	testRWObserverWithFilter(t, req, rewardEvents, rewardEvents)
}

func testObserveRewardsResponsesWithAssetFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveRewardsRequest{AssetId: "asset1"}
	rewardEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 1, "party2", "2", "asset2", num.NewUint(200), num.DecimalFromFloat(0.2), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset1", num.NewUint(300), num.DecimalFromFloat(0.3), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), num.DecimalFromFloat(0.4), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 4, "party2", "5", "asset2", num.NewUint(500), num.DecimalFromFloat(0.5), types.AccountTypeMakerFeeReward, "123"),
	}
	expectedEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset1", num.NewUint(300), num.DecimalFromFloat(0.3), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), num.DecimalFromFloat(0.4), types.AccountTypeMakerFeeReward, "123"),
	}

	testRWObserverWithFilter(t, req, rewardEvents, expectedEvents)
}

func testObserveRewardsResponsesWithPartyFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveRewardsRequest{Party: "party1"}
	rewardEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 1, "party2", "2", "asset2", num.NewUint(200), num.DecimalFromFloat(0.2), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset1", num.NewUint(300), num.DecimalFromFloat(0.3), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), num.DecimalFromFloat(0.4), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 4, "party2", "5", "asset2", num.NewUint(500), num.DecimalFromFloat(0.5), types.AccountTypeMakerFeeReward, "123"),
	}
	expectedEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, ""),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), num.DecimalFromFloat(0.4), types.AccountTypeMakerFeeReward, ""),
	}
	testRWObserverWithFilter(t, req, rewardEvents, expectedEvents)
}

func testObserveRewardsResponsesWithAssetPartyFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveRewardsRequest{Party: "party1", AssetId: "asset1"}
	rewardEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 1, "party2", "2", "asset2", num.NewUint(200), num.DecimalFromFloat(0.2), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset1", num.NewUint(300), num.DecimalFromFloat(0.3), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), num.DecimalFromFloat(0.4), types.AccountTypeMakerFeeReward, "123"),
		events.NewRewardPayout(ctx, 4, "party2", "5", "asset2", num.NewUint(500), num.DecimalFromFloat(0.5), types.AccountTypeMakerFeeReward, "123"),
	}
	expectedEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), num.DecimalFromFloat(0.1), types.AccountTypeMakerFeeReward, ""),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), num.DecimalFromFloat(0.4), types.AccountTypeMakerFeeReward, ""),
	}

	testRWObserverWithFilter(t, req, rewardEvents, expectedEvents)
}

func testRWObserverWithFilter(t *testing.T, req *apipb.ObserveRewardsRequest, evts []*events.RewardPayout, expectedEvents []*events.RewardPayout) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	// we need to subscribe to the stream prior to publishing the events
	stream, err := client.ObserveRewards(ctx, req)
	assert.NoError(t, err)

	// wait until the transfer response has subscribed before sending events
	err = waitForRwSubsription(ctx, server)
	require.NoError(t, err)

	for _, evt := range evts {
		server.broker.Send(evt)
	}

	var i = 0
	for i < len(expectedEvents) {
		resp, err := stream.Recv()

		// Check if the stream has finished
		if err == io.EOF {
			break
		}

		require.NotNil(t, resp)
		require.Equal(t, expectedEvents[i].Party, resp.Reward.PartyId)
		require.Equal(t, expectedEvents[i].Asset, resp.Reward.AssetId)
		require.Equal(t, expectedEvents[i].Amount.String(), resp.Reward.Amount)
		// require.Equal(t, expectedEvents[i].PercentageOfTotalReward, resp.Reward.PercentageOfTotal)
		require.Equal(t, expectedEvents[i].EpochSeq, strconv.Itoa(int(resp.Reward.Epoch)))
		i++
	}
}
