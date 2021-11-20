package rewards

import (
	"context"
	"sort"
	"time"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/protobuf/proto"
)

func (e *Engine) Name() types.CheckpointName {
	return types.PendingRewardsCheckpoint
}

func (e *Engine) partyAmountMapToSlice(partyAmount map[string]*num.Uint) []*checkpoint.PartyAmount {
	parties := make([]string, 0, len(partyAmount))
	for party := range partyAmount {
		parties = append(parties, party)
	}
	sort.Strings(parties)
	res := make([]*checkpoint.PartyAmount, 0, len(parties))
	for _, party := range parties {
		res = append(res, &checkpoint.PartyAmount{Party: party, Amount: partyAmount[party].String()})
	}
	return res
}

func (e *Engine) rewardPayoutsToProto(t time.Time, payouts []*payout) *checkpoint.RewardPayout {
	rp := &checkpoint.RewardPayout{PayoutTime: t.UnixNano()}

	rewardsPayouts := make([]*checkpoint.PendingRewardPayout, 0, len(payouts))
	for _, p := range payouts {
		rewardsPayouts = append(rewardsPayouts, &checkpoint.PendingRewardPayout{
			FromAccount: p.fromAccount,
			Asset:       p.asset,
			EpochSeq:    p.epochSeq,
			TotalReward: p.totalReward.String(),
			Timestamp:   p.timestamp,
			PartyAmount: e.partyAmountMapToSlice(p.partyToAmount),
		})
	}
	rp.RewardsPayout = rewardsPayouts
	return rp
}

func (e *Engine) payoutsFromProto(protoPayouts []*checkpoint.PendingRewardPayout) []*payout {
	payouts := make([]*payout, 0, len(protoPayouts))
	for _, protoPayout := range protoPayouts {
		totalReward, _ := num.UintFromString(protoPayout.TotalReward, 10)
		po := &payout{
			fromAccount:   protoPayout.FromAccount,
			asset:         protoPayout.Asset,
			totalReward:   totalReward,
			epochSeq:      protoPayout.EpochSeq,
			timestamp:     protoPayout.Timestamp,
			partyToAmount: make(map[string]*num.Uint, len(protoPayout.PartyAmount)),
		}
		for _, partyAmount := range protoPayout.PartyAmount {
			amount, _ := num.UintFromString(partyAmount.Amount, 10)
			po.partyToAmount[partyAmount.Party] = amount
		}
		payouts = append(payouts, po)
	}
	return payouts
}

func (e *Engine) Checkpoint() ([]byte, error) {
	times := make([]time.Time, 0, len(e.pendingPayouts))
	for t := range e.pendingPayouts {
		times = append(times, t)
	}
	sort.SliceStable(times, func(i, j int) bool { return times[i].Before(times[j]) })

	payoutAtTimeCP := make([]*checkpoint.RewardPayout, 0, len(times))
	for _, t := range times {
		payoutAtTimeCP = append(payoutAtTimeCP, e.rewardPayoutsToProto(t, e.pendingPayouts[t]))
	}

	rewardCP := &checkpoint.Rewards{Rewards: payoutAtTimeCP}
	return proto.Marshal(rewardCP)
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	cp := &checkpoint.Rewards{}
	if err := proto.Unmarshal(data, cp); err != nil {
		return err
	}

	e.pendingPayouts = make(map[time.Time][]*payout, len(cp.Rewards))
	for _, payoutsAtTime := range cp.Rewards {
		e.pendingPayouts[time.Unix(0, payoutsAtTime.PayoutTime)] = e.payoutsFromProto(payoutsAtTime.RewardsPayout)
	}

	return nil
}
