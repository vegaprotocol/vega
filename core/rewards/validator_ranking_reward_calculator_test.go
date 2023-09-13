package rewards

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/require"
)

func TestCalculateRewardsForValidators(t *testing.T) {
	tm := time.Now()
	// no account balance
	require.Nil(t, calculateRewardsForValidators("1", "zohar", "123", num.UintZero(), tm, []*types.PartyContributionScore{{Party: "z1", Score: num.DecimalFromFloat(0.1)}}, 0))
	// no contributions
	require.Nil(t, calculateRewardsForValidators("1", "zohar", "123", num.NewUint(100), tm, []*types.PartyContributionScore{}, 0))
	// one contributed
	po := calculateRewardsForValidators("1", "zohar-asset", "123", num.NewUint(100), tm, []*types.PartyContributionScore{{Party: "z1", Score: num.DecimalFromFloat(0.1)}}, 1)
	require.NotNil(t, po)
	require.Equal(t, "100", po.totalReward.String())
	require.Equal(t, uint64(1), po.lockedForEpochs)
	require.Equal(t, 1, len(po.partyToAmount))
	require.Equal(t, "100", po.partyToAmount["z1"].String())
	require.Equal(t, "zohar-asset", po.asset)
	require.Equal(t, "123", po.fromAccount)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, tm.Unix(), po.timestamp)

	// 3 contributions
	// z1 - 0.1/0.8 = 0.125 => 125
	// z2 - 0.5/0.8 = 0.625 => 625
	// z3 - 0.2/0.8 = 0.250 => 250
	po = calculateRewardsForValidators("1", "zohar-asset", "123", num.NewUint(1000), tm, []*types.PartyContributionScore{{Party: "z1", Score: num.DecimalFromFloat(0.1)}, {Party: "z2", Score: num.DecimalFromFloat(0.5)}, {Party: "z3", Score: num.DecimalFromFloat(0.2)}}, 1)
	require.NotNil(t, po)
	require.Equal(t, "1000", po.totalReward.String())
	require.Equal(t, uint64(1), po.lockedForEpochs)
	require.Equal(t, 3, len(po.partyToAmount))
	require.Equal(t, "125", po.partyToAmount["z1"].String())
	require.Equal(t, "625", po.partyToAmount["z2"].String())
	require.Equal(t, "250", po.partyToAmount["z3"].String())
	require.Equal(t, "zohar-asset", po.asset)
	require.Equal(t, "123", po.fromAccount)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, tm.Unix(), po.timestamp)
}
