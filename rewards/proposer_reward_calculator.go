package rewards

import (
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// calculateRewardForProposers calculates the reward given to proposers of markets that crossed the trading threshold for the first time.
func calculateRewardForProposers(epochSeq, asset, accountID string, rewardType types.AccountType, balance *num.Uint, proposers []string, timestamp time.Time) *payout {
	if len(proposers) <= 0 || balance.IsZero() || balance.IsNegative() {
		return nil
	}

	po := &payout{
		asset:         asset,
		fromAccount:   accountID,
		epochSeq:      epochSeq,
		timestamp:     timestamp.Unix(),
		partyToAmount: map[string]*num.Uint{},
	}
	total := num.Zero()
	rewardBalance := balance.ToDecimal()
	proposerBonus := rewardBalance.Div(num.DecimalFromInt64(int64(len(proposers))))
	for _, p := range proposers {
		partyReward, _ := num.UintFromDecimal(proposerBonus)
		if !partyReward.IsZero() {
			po.partyToAmount[p] = partyReward
			total.AddSum(partyReward)
		}
	}
	po.totalReward = total
	if total.IsZero() {
		return nil
	}
	return po
}
