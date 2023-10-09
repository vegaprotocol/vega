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

	vega "code.vegaprotocol.io/vega/protos/vega"
)

type updateMarketStateResolver VegaResolverRoot

func (r *updateMarketStateResolver) Market(ctx context.Context, obj *vega.UpdateMarketState) (*vega.Market, error) {
	return r.r.getMarketByID(ctx, obj.Changes.MarketId)
}

func (r *updateMarketStateResolver) UpdateType(ctx context.Context, obj *vega.UpdateMarketState) (MarketUpdateType, error) {
	switch obj.Changes.UpdateType {
	case vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_TERMINATE:
		return MarketUpdateTypeMarketStateUpdateTypeTerminate, nil
	case vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_SUSPEND:
		return MarketUpdateTypeMarketStateUpdateTypeSuspend, nil
	case vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_RESUME:
		return MarketUpdateTypeMarketStateUpdateTypeResume, nil
	default:
		return MarketUpdateTypeMarketStateUpdateTypeUnspecified, nil
	}
}

func (urpd *updateMarketStateResolver) Price(ctx context.Context, obj *vega.UpdateMarketState) (*string, error) {
	return obj.Changes.Price, nil
}
