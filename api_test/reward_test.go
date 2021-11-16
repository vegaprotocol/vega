package api_test

import (
	"context"
	"io"
	"strconv"
	"testing"

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
	req := &apipb.ObserveRewardDetailsRequest{}
	rewardEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), 0.1),
		events.NewRewardPayout(ctx, 1, "party2", "2", "asset2", num.NewUint(200), 0.2),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset2", num.NewUint(300), 0.3),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset2", num.NewUint(400), 0.4),
		events.NewRewardPayout(ctx, 4, "party2", "5", "asset2", num.NewUint(500), 0.5),
	}

	testRWObserverWithFilter(t, req, rewardEvents, rewardEvents)
}

func testObserveRewardsResponsesWithAssetFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveRewardDetailsRequest{AssetId: "asset1"}
	rewardEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), 0.1),
		events.NewRewardPayout(ctx, 1, "party2", "2", "asset2", num.NewUint(200), 0.2),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset1", num.NewUint(300), 0.3),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), 0.4),
		events.NewRewardPayout(ctx, 4, "party2", "5", "asset2", num.NewUint(500), 0.5),
	}
	expectedEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), 0.1),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset1", num.NewUint(300), 0.3),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), 0.4),
	}

	testRWObserverWithFilter(t, req, rewardEvents, expectedEvents)
}

func testObserveRewardsResponsesWithPartyFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveRewardDetailsRequest{Party: "party1"}
	rewardEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), 0.1),
		events.NewRewardPayout(ctx, 1, "party2", "2", "asset2", num.NewUint(200), 0.2),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset1", num.NewUint(300), 0.3),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), 0.4),
		events.NewRewardPayout(ctx, 4, "party2", "5", "asset2", num.NewUint(500), 0.5),
	}
	expectedEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), 0.1),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), 0.4),
	}
	testRWObserverWithFilter(t, req, rewardEvents, expectedEvents)
}

func testObserveRewardsResponsesWithAssetPartyFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveRewardDetailsRequest{Party: "party1", AssetId: "asset1"}
	rewardEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), 0.1),
		events.NewRewardPayout(ctx, 1, "party2", "2", "asset2", num.NewUint(200), 0.2),
		events.NewRewardPayout(ctx, 2, "party3", "3", "asset1", num.NewUint(300), 0.3),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), 0.4),
		events.NewRewardPayout(ctx, 4, "party2", "5", "asset2", num.NewUint(500), 0.5),
	}
	expectedEvents := []*events.RewardPayout{
		events.NewRewardPayout(ctx, 0, "party1", "1", "asset1", num.NewUint(100), 0.1),
		events.NewRewardPayout(ctx, 3, "party1", "4", "asset1", num.NewUint(400), 0.4),
	}

	testRWObserverWithFilter(t, req, rewardEvents, expectedEvents)
}

func testRWObserverWithFilter(t *testing.T, req *apipb.ObserveRewardDetailsRequest, evts []*events.RewardPayout, expectedEvents []*events.RewardPayout) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	// we need to subscribe to the stream prior to publishing the events
	stream, err := client.ObserveRewardDetails(ctx, req)
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
		require.Equal(t, expectedEvents[i].Party, resp.RewardDetails.PartyId)
		require.Equal(t, expectedEvents[i].Asset, resp.RewardDetails.AssetId)
		require.Equal(t, expectedEvents[i].Amount.String(), resp.RewardDetails.Amount)
		require.Equal(t, expectedEvents[i].PercentageOfTotalReward[:7], resp.RewardDetails.PercentageOfTotal)
		require.Equal(t, expectedEvents[i].EpochSeq, strconv.Itoa(int(resp.RewardDetails.Epoch)))
		i++
	}
}
