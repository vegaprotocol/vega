package rewards

import (
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// calculateRewardsByContribution calculates the reward based on the fee contribution (whether paid or received) of the parties in the asset
func calculateRewardsByContribution(epochSeq, asset, accountID string, rewardType types.AccountType, balance *num.Uint, participation []*types.FeePartyScore, timestamp time.Time) *payout {
	po := &payout{
		asset:         asset,
		fromAccount:   accountID,
		epochSeq:      epochSeq,
		timestamp:     timestamp.Unix(),
		partyToAmount: map[string]*num.Uint{},
	}
	total := num.Zero()
	rewardBalance := balance.ToDecimal()
	for _, p := range participation {
		partyReward, _ := num.UintFromDecimal(rewardBalance.Mul(p.Score))
		if !partyReward.IsZero() {
			po.partyToAmount[p.Party] = partyReward
			total.AddSum(partyReward)
		}
	}
	po.totalReward = total
	if total.IsZero() {
		return nil
	}
	return po
}
