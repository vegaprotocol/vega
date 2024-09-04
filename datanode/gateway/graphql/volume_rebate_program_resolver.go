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

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type volumeRebateProgramResolver VegaResolverRoot

// BenefitTiers implements VolumeRebateProgramResolver.
func (r *volumeRebateProgramResolver) BenefitTiers(ctx context.Context, obj *v2.VolumeRebateProgram) ([]*VolumeRebateBenefitTier, error) {
	tiers := make([]*VolumeRebateBenefitTier, 0, len(obj.BenefitTiers))
	for _, tier := range obj.BenefitTiers {
		tiers = append(tiers, &VolumeRebateBenefitTier{
			MinimumPartyMakerVolumeFraction: tier.MinimumPartyMakerVolumeFraction,
			AdditionalMakerRebate:           tier.AdditionalMakerRebate,
			TierNumber:                      ptr.From(int(ptr.UnBox(tier.TierNumber))),
		})
	}
	return tiers, nil
}

func (r *volumeRebateProgramResolver) Version(_ context.Context, obj *v2.VolumeRebateProgram) (int, error) {
	if obj.Version > math.MaxInt {
		return 0, errors.New("version is too large")
	}

	return int(obj.Version), nil
}

func (r *volumeRebateProgramResolver) WindowLength(_ context.Context, obj *v2.VolumeRebateProgram) (int, error) {
	if obj.WindowLength > math.MaxInt {
		return 0, errors.New("window length is too large")
	}

	return int(obj.WindowLength), nil
}
