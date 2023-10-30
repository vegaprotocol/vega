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
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type perpetualResolver VegaResolverRoot

func (r *perpetualResolver) SettlementAsset(ctx context.Context, obj *vega.Perpetual) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.SettlementAsset)
}

func (r *perpetualResolver) DataSourceSpecForSettlementSchedule(ctx context.Context, obj *vega.Perpetual) (*DataSourceSpec, error) {
	return resolveDataSourceSpec(obj.DataSourceSpecForSettlementSchedule), nil
}

func (r *perpetualResolver) DataSourceSpecForSettlementData(ctx context.Context, obj *vega.Perpetual) (*DataSourceSpec, error) {
	return resolveDataSourceSpec(obj.DataSourceSpecForSettlementData), nil
}

func (r *perpetualResolver) DataSourceSpecBinding(ctx context.Context, obj *vega.Perpetual) (*DataSourceSpecPerpetualBinding, error) {
	return &DataSourceSpecPerpetualBinding{
		SettlementDataProperty:     obj.DataSourceSpecBinding.SettlementDataProperty,
		SettlementScheduleProperty: obj.DataSourceSpecBinding.SettlementScheduleProperty,
	}, nil
}

type perpetualProductResolver VegaResolverRoot

func (r *perpetualProductResolver) SettlementAsset(ctx context.Context, obj *vega.PerpetualProduct) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.SettlementAsset)
}

func (r *perpetualProductResolver) DataSourceSpecBinding(ctx context.Context, obj *vega.PerpetualProduct) (*DataSourceSpecPerpetualBinding, error) {
	return &DataSourceSpecPerpetualBinding{
		SettlementDataProperty:     obj.DataSourceSpecBinding.SettlementDataProperty,
		SettlementScheduleProperty: obj.DataSourceSpecBinding.SettlementScheduleProperty,
	}, nil
}

func (r *perpetualProductResolver) DataSourceSpecForSettlementData(_ context.Context, obj *vegapb.PerpetualProduct) (*vegapb.DataSourceDefinition, error) {
	if obj.DataSourceSpecForSettlementData == nil {
		return nil, nil
	}
	return resolveDataSourceDefinition(obj.DataSourceSpecForSettlementData), nil
}

func (r *perpetualProductResolver) DataSourceSpecForSettlementSchedule(_ context.Context, obj *vegapb.PerpetualProduct) (*vegapb.DataSourceDefinition, error) {
	if obj.DataSourceSpecForSettlementSchedule == nil {
		return nil, nil
	}
	return resolveDataSourceDefinition(obj.DataSourceSpecForSettlementSchedule), nil
}
