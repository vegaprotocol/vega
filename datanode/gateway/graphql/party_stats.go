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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type partyDiscountStatsResolver VegaResolverRoot

func (m *partyDiscountStatsResolver) BaseMakerFee(_ context.Context, obj *v2.MarketFees) (string, error) {
	return obj.BaseMakerRebate, nil
}

func (r *partyDiscountStatsResolver) VolumeDiscountTier(_ context.Context, obj *v2.GetPartyDiscountStatsResponse) (int, error) {
	return int(obj.VolumeDiscountTier), nil
}

func (r *partyDiscountStatsResolver) VolumeRebateTier(_ context.Context, obj *v2.GetPartyDiscountStatsResponse) (int, error) {
	return int(obj.VolumeRebateTier), nil
}

func (r *partyDiscountStatsResolver) ReferralDiscountTier(_ context.Context, obj *v2.GetPartyDiscountStatsResponse) (int, error) {
	return int(obj.ReferralDiscountTier), nil
}
