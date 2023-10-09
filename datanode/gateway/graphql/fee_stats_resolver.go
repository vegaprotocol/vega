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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	partyAmountResolver              VegaResolverRoot
	referralSetFeeStatsResolver      VegaResolverRoot
	referrerRewardsGeneratedResolver VegaResolverRoot
)

func (r *partyAmountResolver) PartyID(_ context.Context, obj *eventspb.PartyAmount) (string, error) {
	return obj.Party, nil
}

func (r *referralSetFeeStatsResolver) MarketID(ctx context.Context, obj *eventspb.FeeStats) (string, error) {
	return obj.Market, nil
}

func (r *referralSetFeeStatsResolver) AssetID(ctx context.Context, obj *eventspb.FeeStats) (string, error) {
	return obj.Asset, nil
}

func (r *referralSetFeeStatsResolver) Epoch(ctx context.Context, obj *eventspb.FeeStats) (int, error) {
	return int(obj.EpochSeq), nil
}

func (r *referrerRewardsGeneratedResolver) ReferrerID(ctx context.Context, obj *eventspb.ReferrerRewardsGenerated) (string, error) {
	return obj.Referrer, nil
}
