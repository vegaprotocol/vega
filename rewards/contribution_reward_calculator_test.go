package rewards

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
)

func TestCalculateRewardsByContribution(t *testing.T) {
	require.Nil(t, calculateRewardsByContribution("1", "ETH", "FROM_ACCOUNT", types.AccountTypeMakerFeeReward, num.Zero(), []*types.FeePartyScore{{Party: "party1", Score: num.DecimalFromFloat(0.2)}, {Party: "party2", Score: num.DecimalFromFloat(0.8)}}, time.Now()))
	require.Nil(t, calculateRewardsByContribution("1", "ETH", "FROM_ACCOUNT", types.AccountTypeMakerFeeReward, num.NewUint(1000), []*types.FeePartyScore{}, time.Now()))

	po := calculateRewardsByContribution("1", "ETH", "FROM_ACCOUNT", types.AccountTypeMakerFeeReward, num.NewUint(1000), []*types.FeePartyScore{{Party: "party1", Score: num.DecimalFromFloat(0.2)}, {Party: "party2", Score: num.DecimalFromFloat(0.8)}}, time.Now())
	require.Equal(t, "ETH", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "FROM_ACCOUNT", po.fromAccount)
	require.Equal(t, "200", po.partyToAmount["party1"].String())
	require.Equal(t, "800", po.partyToAmount["party2"].String())
	require.Equal(t, 2, len(po.partyToAmount))
}
