package execution_test

import (
	"bytes"
	"context"
	"testing"

	snapshotpb "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
)

type EligibilityChecker struct{}

func (e *EligibilityChecker) IsEligibleForProposerBonus(marketID string, volumeTraded *num.Uint) bool {
	return volumeTraded.GT(num.NewUint(5000))
}

func TestMarketTracker(t *testing.T) {
	tracker := execution.NewMarketTracker()
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	tracker.MarketProposed("market1", "me")
	tracker.MarketProposed("market2", "me2")

	require.Equal(t, 0, len(tracker.GetAndResetEligibleProposers()))

	tracker.AddValueTraded("market1", num.NewUint(1000))
	require.Equal(t, 0, len(tracker.GetAndResetEligibleProposers()))

	tracker.AddValueTraded("market2", num.NewUint(4000))
	require.Equal(t, 0, len(tracker.GetAndResetEligibleProposers()))

	tracker.AddValueTraded("market2", num.NewUint(1001))
	tracker.AddValueTraded("market1", num.NewUint(4001))

	eligible := tracker.GetAndResetEligibleProposers()
	require.Equal(t, 2, len(eligible))
	require.Equal(t, "me", eligible[0])
	require.Equal(t, "me2", eligible[1])

	// ask again and expect nothing to be returned
	require.Equal(t, 0, len(tracker.GetAndResetEligibleProposers()))

	// take a snapshot
	key := (&types.PayloadMarketTracker{}).Key()
	hash1, err := tracker.GetHash(key)
	require.NoError(t, err)

	state1, _, err := tracker.GetState(key)
	require.NoError(t, err)

	trackerLoad := execution.NewMarketTracker()
	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state1, &pl))

	trackerLoad.LoadState(context.Background(), types.PayloadFromProto(&pl))

	hash2, err := trackerLoad.GetHash(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(hash1, hash2))

	state2, _, err := trackerLoad.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state1, state2))
}

func TestMarketTrackerStateChange(t *testing.T) {
	key := (&types.PayloadMarketTracker{}).Key()

	tracker := execution.NewMarketTracker()
	tracker.SetEligibilityChecker(&EligibilityChecker{})

	hash1, err := tracker.GetHash(key)
	require.NoError(t, err)

	tracker.MarketProposed("market1", "me")
	tracker.MarketProposed("market2", "me2")

	hash2, err := tracker.GetHash(key)
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash2)

	tracker.AddValueTraded("market1", num.NewUint(1000))
	require.Equal(t, 0, len(tracker.GetAndResetEligibleProposers()))

	hash3, err := tracker.GetHash(key)
	require.NoError(t, err)
	require.NotEqual(t, hash2, hash3)
}
