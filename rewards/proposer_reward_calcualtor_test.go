package rewards

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
)

func TestProposerBonusCalculator(t *testing.T) {
	now := time.Now()
	require.Nil(t, calculateRewardForProposers("1", "asset", "123456", types.AccountTypeMarketProposerReward, num.Zero(), []string{"mememe"}, now))
	require.Nil(t, calculateRewardForProposers("1", "asset", "123456", types.AccountTypeMarketProposerReward, num.NewUint(10000), []string{}, now))

	po := calculateRewardForProposers("1", "asset", "123456", types.AccountTypeMarketProposerReward, num.NewUint(9000), []string{"p1", "p2", "p3"}, now)
	require.Equal(t, "asset", po.asset)
	require.Equal(t, "1", po.epochSeq)
	require.Equal(t, "123456", po.fromAccount)
	require.Equal(t, now.Unix(), po.timestamp)
	require.Equal(t, "3000", po.partyToAmount["p1"].String())
	require.Equal(t, "3000", po.partyToAmount["p2"].String())
	require.Equal(t, "3000", po.partyToAmount["p3"].String())
	require.Equal(t, 3, len(po.partyToAmount))
}
