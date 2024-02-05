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
	vega "code.vegaprotocol.io/vega/protos/vega"
)

type myMarginEstimateResolver VegaResolverRoot

// BestCase implements MarginEstimateResolver.
func (me *myMarginEstimateResolver) BestCase(ctx context.Context, obj *v2.MarginEstimate) (*AbstractMarginLevels, error) {
	if obj == nil {
		return nil, nil
	}

	return me.marginEstimateToAbstract(ctx, obj.BestCase)
}

// WorstCase implements MarginEstimateResolver.
func (me *myMarginEstimateResolver) WorstCase(ctx context.Context, obj *v2.MarginEstimate) (*AbstractMarginLevels, error) {
	return me.marginEstimateToAbstract(ctx, obj.WorstCase)
}

func (me *myMarginEstimateResolver) marginEstimateToAbstract(ctx context.Context, obj *vega.MarginLevels) (*AbstractMarginLevels, error) {
	market, err := me.r.getMarketByID(ctx, obj.MarketId)
	if err != nil {
		return nil, err
	}

	asset, err := me.r.getAssetByID(ctx, obj.Asset)
	if err != nil {
		return nil, err
	}
	return &AbstractMarginLevels{
		Market:                 market,
		Asset:                  asset,
		MaintenanceLevel:       obj.MaintenanceMargin,
		SearchLevel:            obj.SearchLevel,
		InitialLevel:           obj.InitialMargin,
		OrderMarginLevel:       obj.OrderMargin,
		CollateralReleaseLevel: obj.CollateralReleaseLevel,
		MarginMode:             obj.MarginMode,
		MarginFactor:           obj.MarginFactor,
	}, nil
}
