// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	vgreflect "code.vegaprotocol.io/vega/libs/reflect"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type VestingBenefitTiers struct {
	Tiers []*VestingBenefitTier
}

type VestingBenefitTier struct {
	MinimumQuantumBalance *num.Uint
	RewardMultiplier      num.Decimal
}

func (a *VestingBenefitTiers) Clone() *VestingBenefitTiers {
	out := &VestingBenefitTiers{}

	for _, v := range a.Tiers {
		out.Tiers = append(out.Tiers, &VestingBenefitTier{
			MinimumQuantumBalance: v.MinimumQuantumBalance.Clone(),
			RewardMultiplier:      v.RewardMultiplier,
		})
	}

	return out
}

func VestingBenefitTiersFromUntypedProto(v interface{}) (*VestingBenefitTiers, error) {
	ptiers, err := toVestingBenefitTier(v)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert untyped proto to VestingBenefitTiers proto: %w", err)
	}

	tiers, err := VestingBenefitTiersFromProto(ptiers)
	if err != nil {
		return nil, fmt.Errorf("couldn't build EthereumConfig: %w", err)
	}

	return tiers, nil
}

func VestingBenefitTiersFromProto(ptiers *proto.VestingBenefitTiers) (*VestingBenefitTiers, error) {
	err := CheckVestingBenefitTiers(ptiers)
	if err != nil {
		return nil, err
	}

	tiers := &VestingBenefitTiers{}
	for _, v := range ptiers.Tiers {
		balance, _ := num.UintFromString(v.MinimumQuantumBalance, 10)
		tiers.Tiers = append(tiers.Tiers, &VestingBenefitTier{
			MinimumQuantumBalance: balance,
			RewardMultiplier:      num.MustDecimalFromString(v.RewardMultiplier),
		})
	}

	return tiers, nil
}

func CheckUntypedVestingBenefitTier(v interface{}) error {
	tiers, err := toVestingBenefitTier(v)
	if err != nil {
		return err
	}

	return CheckVestingBenefitTiers(tiers)
}

// CheckEthereumConfig verifies the proto.EthereumConfig is valid.
func CheckVestingBenefitTiers(ptiers *proto.VestingBenefitTiers) error {
	if len(ptiers.Tiers) <= 0 {
		return ErrNoTierSet
	}

	minQuantumVolumeSet := map[num.Uint]struct{}{}

	for i, v := range ptiers.Tiers {
		minQuantumBalance, underflow := num.UintFromString(v.MinimumQuantumBalance, 10)
		if underflow {
			return fmt.Errorf("invalid %d.minimum_quantum_balance", i)
		}
		if _, ok := minQuantumVolumeSet[*minQuantumBalance]; ok {
			return fmt.Errorf("duplicate minimum_activity_streak entry for: %s", v.MinimumQuantumBalance)
		}
		minQuantumVolumeSet[*minQuantumBalance] = struct{}{}

		if _, err := num.DecimalFromString(v.RewardMultiplier); err != nil {
			return fmt.Errorf("%d.reward_multiplier, invalid decimal: %w", i, err)
		}
	}

	return nil
}

func toVestingBenefitTier(v interface{}) (*proto.VestingBenefitTiers, error) {
	cfg, ok := v.(*proto.VestingBenefitTiers)
	if !ok {
		return nil, fmt.Errorf("type \"%s\" is not a *VestingBenefitTiers proto", vgreflect.TypeName(v))
	}

	return cfg, nil
}