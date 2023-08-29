package rewards

import (
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func calculateRewardsForValidators(epochSeq, asset, accountID string, balance *num.Uint, timestamp time.Time, rankingScoreContributions []*types.PartyContibutionScore, lockForEpochs uint64) *payout {
	if balance.IsZero() || balance.IsNegative() {
		return nil
	}

	total := num.DecimalZero()
	for _, pcs := range rankingScoreContributions {
		total = total.Add(pcs.Score)
	}

	if total.IsZero() {
		return nil
	}
	normalise(rankingScoreContributions, total)

	po := &payout{
		asset:           asset,
		fromAccount:     accountID,
		epochSeq:        epochSeq,
		timestamp:       timestamp.Unix(),
		partyToAmount:   map[string]*num.Uint{},
		lockedForEpochs: lockForEpochs,
		totalReward:     num.UintZero(),
	}
	for _, pcs := range rankingScoreContributions {
		r, _ := num.UintFromDecimal(pcs.Score.Mul(balance.ToDecimal()))
		po.partyToAmount[pcs.Party] = r
		po.totalReward.AddSum(r)
	}
	return po
}
