// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
