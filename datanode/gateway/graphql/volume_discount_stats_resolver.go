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

	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type volumeDiscountStatsResolver VegaResolverRoot

// DiscountFactors implements VolumeDiscountStatsResolver.
func (v *volumeDiscountStatsResolver) DiscountFactors(ctx context.Context, obj *v2.VolumeDiscountStats) (*DiscountFactors, error) {
	infra, err := num.DecimalFromString(obj.DiscountFactors.InfrastructureDiscountFactor)
	if err != nil {
		return nil, err
	}
	maker, err := num.DecimalFromString(obj.DiscountFactors.MakerDiscountFactor)
	if err != nil {
		return nil, err
	}
	liq, err := num.DecimalFromString(obj.DiscountFactors.LiquidityDiscountFactor)
	if err != nil {
		return nil, err
	}
	return &DiscountFactors{
		InfrastructureFactor: infra.String(),
		MakerFactor:          maker.String(),
		LiquidityFactor:      liq.String(),
	}, nil
}

func (v *volumeDiscountStatsResolver) AtEpoch(_ context.Context, obj *v2.VolumeDiscountStats) (int, error) {
	if obj.AtEpoch > math.MaxInt {
		return 0, errors.New("at_epoch is too large")
	}

	return int(obj.AtEpoch), nil
}
