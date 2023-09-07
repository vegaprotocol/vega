package gql

import (
	"context"

	v1 "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type partyActivityStreakResolver VegaResolverRoot

func (p *partyActivityStreakResolver) ActiveFor(ctx context.Context, obj *v1.PartyActivityStreak) (int, error) {
	return int(obj.ActiveFor), nil
}

func (p *partyActivityStreakResolver) InactiveFor(ctx context.Context, obj *v1.PartyActivityStreak) (int, error) {
	return int(obj.InactiveFor), nil
}

func (p *partyActivityStreakResolver) RewardDistributionMultiplier(ctx context.Context, obj *v1.PartyActivityStreak) (string, error) {
	return obj.RewardDistributionActivityMultiplier, nil
}

func (p *partyActivityStreakResolver) RewardVestingMultiplier(ctx context.Context, obj *v1.PartyActivityStreak) (string, error) {
	return obj.RewardVestingActivityMultiplier, nil
}

func (p *partyActivityStreakResolver) Epoch(ctx context.Context, obj *v1.PartyActivityStreak) (int, error) {
	return int(obj.Epoch), nil
}
