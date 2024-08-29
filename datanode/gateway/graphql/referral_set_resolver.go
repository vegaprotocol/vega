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
	"errors"
	"math"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type referralSetResolver VegaResolverRoot

func (t referralSetResolver) TotalMembers(_ context.Context, obj *v2.ReferralSet) (int, error) {
	return int(obj.TotalMembers), nil
}

type referralSetRefereeResolver VegaResolverRoot

func (r *referralSetRefereeResolver) AtEpoch(ctx context.Context, obj *v2.ReferralSetReferee) (int, error) {
	if obj == nil {
		return 0, nil
	}

	return int(obj.AtEpoch), nil
}

func (r *referralSetRefereeResolver) RefereeID(ctx context.Context, obj *v2.ReferralSetReferee) (string, error) {
	return obj.Referee, nil
}

type referralSetStatsResolver VegaResolverRoot

// DiscountFactors implements ReferralSetStatsResolver.
func (r *referralSetStatsResolver) DiscountFactors(ctx context.Context, obj *v2.ReferralSetStats) (*DiscountFactors, error) {
	return &DiscountFactors{
		InfrastructureFactor: obj.DiscountFactors.InfrastructureDiscountFactor,
		MakerFactor:          obj.DiscountFactors.MakerDiscountFactor,
		LiquidityFactor:      obj.DiscountFactors.LiquidityDiscountFactor,
	}, nil
}

// RewardFactors implements ReferralSetStatsResolver.
func (r *referralSetStatsResolver) RewardFactors(ctx context.Context, obj *v2.ReferralSetStats) (*RewardFactors, error) {
	return &RewardFactors{
		InfrastructureFactor: obj.RewardFactors.InfrastructureRewardFactor,
		MakerFactor:          obj.RewardFactors.MakerRewardFactor,
		LiquidityFactor:      obj.RewardFactors.LiquidityRewardFactor,
	}, nil
}

// RewardsFactorsMultiplier implements ReferralSetStatsResolver.
func (r *referralSetStatsResolver) RewardsFactorsMultiplier(ctx context.Context, obj *v2.ReferralSetStats) (*RewardFactors, error) {
	return &RewardFactors{
		InfrastructureFactor: obj.RewardsFactorsMultiplier.InfrastructureRewardFactor,
		MakerFactor:          obj.RewardsFactorsMultiplier.MakerRewardFactor,
		LiquidityFactor:      obj.RewardsFactorsMultiplier.LiquidityRewardFactor,
	}, nil
}

func (r *referralSetStatsResolver) AtEpoch(_ context.Context, obj *v2.ReferralSetStats) (int, error) {
	if obj.AtEpoch > math.MaxInt {
		return 0, errors.New("at_epoch is too large")
	}

	return int(obj.AtEpoch), nil
}
