package rewards

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/proto"
)

var (
	key = (&types.PayloadRewardsPayout{}).Key()

	hashKeys = []string{
		key,
	}

	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for reward payout snapshot")
)

type rewardsSnapshotState struct {
	changed    bool
	hash       []byte
	serialised []byte
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.RewardSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) serialisePayout() ([]byte, error) {
	payouts := []*types.ScheduledRewardsPayout{}

	for t, p := range e.pendingPayouts {
		pending := make([]*types.RewardsPayout, 0, len(p))
		for _, pp := range p {
			partyAmounts := make([]*types.RewardsPartyAmount, 0, len(pp.partyToAmount))
			for party, amount := range pp.partyToAmount {
				partyAmounts = append(partyAmounts, &types.RewardsPartyAmount{Party: party, Amount: amount})
			}

			sort.SliceStable(partyAmounts, func(i, j int) bool { return partyAmounts[i].Party < partyAmounts[j].Party })
			pending = append(pending, &types.RewardsPayout{
				FromAccount:  pp.fromAccount,
				Asset:        pp.asset,
				EpochSeq:     pp.epochSeq,
				Timestamp:    pp.timestamp,
				TotalReward:  pp.totalReward,
				PartyAmounts: partyAmounts,
			})
		}

		sort.SliceStable(pending, func(i, j int) bool {
			switch strings.Compare(pending[i].FromAccount, pending[j].FromAccount) {
			case -1:
				return true
			case 1:
				return false
			}

			if pending[i].EpochSeq == pending[j].EpochSeq {
				switch strings.Compare(pending[i].Asset, pending[j].Asset) {
				case -1:
					return true
				case 0:
					e.log.Panic("multiple payouts for the same epoch, fromAccount, asset", logging.String("fromAccount", pending[i].FromAccount), logging.String("asset", pending[i].Asset), logging.String("epochSeq", pending[i].EpochSeq))
				default:
					return false
				}
			}

			return pending[i].EpochSeq < pending[j].EpochSeq
		})

		payouts = append(payouts, &types.ScheduledRewardsPayout{
			PayoutTime:    t.UnixNano(),
			RewardsPayout: pending,
		})
	}

	sort.SliceStable(payouts, func(i, j int) bool { return payouts[i].PayoutTime < payouts[j].PayoutTime })
	payload := types.Payload{
		Data: &types.PayloadRewardsPayout{
			RewardsPendingPayouts: &types.RewardsPendingPayouts{
				ScheduledRewardsPayout: payouts,
			},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

// get the serialised form and hash of the given key.
func (e *Engine) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != key {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !e.rss.changed {
		return e.rss.serialised, e.rss.hash, nil
	}

	data, err := e.serialisePayout()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	e.rss.serialised = data
	e.rss.hash = hash
	e.rss.changed = false
	return data, hash, nil
}

func (e *Engine) GetHash(k string) ([]byte, error) {
	_, hash, err := e.getSerialisedAndHash(k)
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, error) {
	state, _, err := e.getSerialisedAndHash(k)
	return state, err
}

func (e *Engine) Snapshot() (map[string][]byte, error) {
	r := make(map[string][]byte, len(hashKeys))
	for _, k := range hashKeys {
		state, err := e.GetState(k)
		if err != nil {
			return nil, err
		}
		r[k] = state
	}
	return r, nil
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) error {
	if e.Namespace() != p.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadRewardsPayout:
		return e.restorePayout(ctx, pl.RewardsPendingPayouts)
	default:
		return types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restorePayout(ctx context.Context, rpp *types.RewardsPendingPayouts) error {
	e.pendingPayouts = map[time.Time][]*payout{}

	for _, srp := range rpp.ScheduledRewardsPayout {
		tt := time.Unix(0, srp.PayoutTime).UTC()
		pots := make([]*payout, 0, len(srp.RewardsPayout))
		partyToAmount := map[string]*num.Uint{}

		for _, po := range srp.RewardsPayout {
			for _, pa := range po.PartyAmounts {
				partyToAmount[pa.Party] = pa.Amount
			}
			pots = append(pots, &payout{
				fromAccount:   po.FromAccount,
				asset:         po.Asset,
				totalReward:   po.TotalReward,
				epochSeq:      po.EpochSeq,
				timestamp:     po.Timestamp,
				partyToAmount: partyToAmount,
			})
		}
		e.pendingPayouts[tt] = pots
	}

	e.rss.changed = true
	return nil
}
