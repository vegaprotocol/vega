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

	"code.vegaprotocol.io/vega/protos/vega"
)

type dispatchStrategyResolver VegaResolverRoot

func (d *dispatchStrategyResolver) DispatchMetric(ctx context.Context, obj *vega.DispatchStrategy) (vega.DispatchMetric, error) {
	return obj.Metric, nil
}

func (d *dispatchStrategyResolver) DispatchMetricAssetID(ctx context.Context, obj *vega.DispatchStrategy) (string, error) {
	return obj.AssetForMetric, nil
}

func (d *dispatchStrategyResolver) MarketIdsInScope(ctx context.Context, obj *vega.DispatchStrategy) ([]string, error) {
	return obj.Markets, nil
}

func (d *dispatchStrategyResolver) WindowLength(ctx context.Context, obj *vega.DispatchStrategy) (int, error) {
	return int(obj.WindowLength), nil
}

func (d *dispatchStrategyResolver) LockPeriod(ctx context.Context, obj *vega.DispatchStrategy) (int, error) {
	return int(obj.LockPeriod), nil
}

func (d *dispatchStrategyResolver) TransferInterval(ctx context.Context, obj *vega.DispatchStrategy) (int, error) {
	if obj.TransferInterval == nil {
		return 1, nil
	}
	return int(*obj.TransferInterval), nil
}
