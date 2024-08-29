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
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

type benefitTierResolver VegaResolverRoot

// ReferralRewardFactors implements BenefitTierResolver.
func (br *benefitTierResolver) ReferralRewardFactors(ctx context.Context, obj *vega.BenefitTier) (*RewardFactors, error) {
	infra, err := num.DecimalFromString(obj.ReferralRewardFactors.InfrastructureRewardFactor)
	if err != nil {
		return nil, err
	}
	maker, err := num.DecimalFromString(obj.ReferralRewardFactors.MakerRewardFactor)
	if err != nil {
		return nil, err
	}
	liq, err := num.DecimalFromString(obj.ReferralRewardFactors.LiquidityRewardFactor)
	if err != nil {
		return nil, err
	}
	return &RewardFactors{
		InfrastructureFactor: infra.String(),
		MakerFactor:          maker.String(),
		LiquidityFactor:      liq.String(),
	}, nil
}

// Referrals implements BenefitTierResolver.
func (br *benefitTierResolver) ReferralDiscountFactors(ctx context.Context, obj *vega.BenefitTier) (*DiscountFactors, error) {
	infra, err := num.DecimalFromString(obj.ReferralDiscountFactors.InfrastructureDiscountFactor)
	if err != nil {
		return nil, err
	}
	maker, err := num.DecimalFromString(obj.ReferralDiscountFactors.MakerDiscountFactor)
	if err != nil {
		return nil, err
	}
	liq, err := num.DecimalFromString(obj.ReferralDiscountFactors.LiquidityDiscountFactor)
	if err != nil {
		return nil, err
	}
	return &DiscountFactors{
		InfrastructureFactor: infra.String(),
		MakerFactor:          maker.String(),
		LiquidityFactor:      liq.String(),
	}, nil
}

func (br *benefitTierResolver) MinimumEpochs(_ context.Context, obj *vega.BenefitTier) (int, error) {
	minEpochs, err := strconv.ParseInt(obj.MinimumEpochs, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse minimum epochs %s: %v", obj.MinimumEpochs, err)
	}

	return int(minEpochs), nil
}

func (v *benefitTierResolver) TierNumber(_ context.Context, obj *vega.BenefitTier) (*int, error) {
	if obj.TierNumber == nil {
		return nil, nil
	}
	i := int(*obj.TierNumber)
	return &i, nil
}

type volumeBenefitTierResolver VegaResolverRoot

// VolumeDiscountFactors implements VolumeBenefitTierResolver.
func (v *volumeBenefitTierResolver) VolumeDiscountFactors(ctx context.Context, obj *vega.VolumeBenefitTier) (*DiscountFactors, error) {
	infra, err := num.DecimalFromString(obj.VolumeDiscountFactors.InfrastructureDiscountFactor)
	if err != nil {
		return nil, err
	}
	maker, err := num.DecimalFromString(obj.VolumeDiscountFactors.MakerDiscountFactor)
	if err != nil {
		return nil, err
	}
	liq, err := num.DecimalFromString(obj.VolumeDiscountFactors.LiquidityDiscountFactor)
	if err != nil {
		return nil, err
	}
	return &DiscountFactors{
		InfrastructureFactor: infra.String(),
		MakerFactor:          maker.String(),
		LiquidityFactor:      liq.String(),
	}, nil
}

func (v *volumeBenefitTierResolver) TierNumber(_ context.Context, obj *vega.VolumeBenefitTier) (*int, error) {
	if obj.TierNumber == nil {
		return nil, nil
	}
	i := int(*obj.TierNumber)
	return &i, nil
}
