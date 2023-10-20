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

package types

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	vgreflect "code.vegaprotocol.io/vega/libs/reflect"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

var ErrNoTierSet = errors.New("no tier set")

type ActivityStreakBenefitTiers struct {
	Tiers []*ActivityStreakBenefitTier
}

type ActivityStreakBenefitTier struct {
	MinimumActivityStreak uint64
	RewardMultiplier      num.Decimal
	VestingMultiplier     num.Decimal
}

func (a *ActivityStreakBenefitTiers) Clone() *ActivityStreakBenefitTiers {
	out := &ActivityStreakBenefitTiers{}

	for _, v := range a.Tiers {
		out.Tiers = append(out.Tiers, &ActivityStreakBenefitTier{
			MinimumActivityStreak: v.MinimumActivityStreak,
			RewardMultiplier:      v.RewardMultiplier,
			VestingMultiplier:     v.VestingMultiplier,
		})
	}

	return out
}

func ActivityStreakBenefitTiersFromUntypedProto(v interface{}) (*ActivityStreakBenefitTiers, error) {
	ptiers, err := toActivityStreakBenefitTier(v)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert untyped proto to ActivityStreakBenefitTiers proto: %w", err)
	}

	tiers, err := ActivityStreakBenefitTiersFromProto(ptiers)
	if err != nil {
		return nil, fmt.Errorf("couldn't build EthereumConfig: %w", err)
	}

	return tiers, nil
}

func ActivityStreakBenefitTiersFromProto(ptiers *proto.ActivityStreakBenefitTiers) (*ActivityStreakBenefitTiers, error) {
	err := CheckActivityStreakBenefitTiers(ptiers)
	if err != nil {
		return nil, err
	}

	tiers := &ActivityStreakBenefitTiers{}
	for _, v := range ptiers.Tiers {
		tiers.Tiers = append(tiers.Tiers, &ActivityStreakBenefitTier{
			MinimumActivityStreak: v.MinimumActivityStreak,
			RewardMultiplier:      num.MustDecimalFromString(v.RewardMultiplier),
			VestingMultiplier:     num.MustDecimalFromString(v.VestingMultiplier),
		})
	}

	return tiers, nil
}

func CheckUntypedActivityStreakBenefitTier(v interface{}) error {
	tiers, err := toActivityStreakBenefitTier(v)
	if err != nil {
		return err
	}

	return CheckActivityStreakBenefitTiers(tiers)
}

// CheckEthereumConfig verifies the proto.EthereumConfig is valid.
func CheckActivityStreakBenefitTiers(ptiers *proto.ActivityStreakBenefitTiers) error {
	if len(ptiers.Tiers) <= 0 {
		return ErrNoTierSet
	}

	activityStreakSet := map[uint64]struct{}{}

	for i, v := range ptiers.Tiers {
		if _, ok := activityStreakSet[v.MinimumActivityStreak]; ok {
			return fmt.Errorf("duplicate minimum_activity_streak entry for: %d", v.MinimumActivityStreak)
		}
		activityStreakSet[v.MinimumActivityStreak] = struct{}{}
		d, err := num.DecimalFromString(v.RewardMultiplier)
		if err != nil {
			return fmt.Errorf("%d.reward_multiplier, invalid decimal: %w", i, err)
		}
		if d.LessThan(num.DecimalOne()) {
			return fmt.Errorf("%d.reward_multiplier, less than 1.0", i)
		}

		d, err = num.DecimalFromString(v.VestingMultiplier)
		if err != nil {
			return fmt.Errorf("%d.vesting_multiplier, invalid decimal: %w", i, err)
		}
		if d.LessThan(num.DecimalOne()) {
			return fmt.Errorf("%d.vesting_multiplier, less than 1.0", i)
		}
	}

	return nil
}

func toActivityStreakBenefitTier(v interface{}) (*proto.ActivityStreakBenefitTiers, error) {
	cfg, ok := v.(*proto.ActivityStreakBenefitTiers)
	if !ok {
		return nil, fmt.Errorf("type \"%s\" is not a *ActivityStreakBenefitTiers proto", vgreflect.TypeName(v))
	}

	return cfg, nil
}
