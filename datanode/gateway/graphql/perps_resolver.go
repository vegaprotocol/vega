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
