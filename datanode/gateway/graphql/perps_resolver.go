package gql

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/protos/vega"
)

type perpetualResolver VegaResolverRoot

func (r *perpetualResolver) SettlementAsset(ctx context.Context, obj *vega.Perpetual) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.SettlementAsset)
}

func (r *perpetualResolver) DataSourceSpecForSettlementSchedule(ctx context.Context, obj *vega.Perpetual) (*DataSourceSpec, error) {
	var status DataSourceSpecStatus

	switch obj.DataSourceSpecForSettlementSchedule.Status {
	case vega.DataSourceSpec_STATUS_ACTIVE:
		status = DataSourceSpecStatusStatusActive
	case vega.DataSourceSpec_STATUS_DEACTIVATED:
		status = DataSourceSpecStatusStatusDeactivated
	default:
		return nil, fmt.Errorf("unknown status: %v", obj.DataSourceSpecForSettlementSchedule.Status)
	}

	return &DataSourceSpec{
		ID:        obj.DataSourceSpecForSettlementSchedule.Id,
		CreatedAt: obj.DataSourceSpecForSettlementSchedule.CreatedAt,
		UpdatedAt: &obj.DataSourceSpecForSettlementSchedule.UpdatedAt,
		Data:      obj.DataSourceSpecForSettlementSchedule.Data,
		Status:    status,
	}, nil
}

func (r *perpetualResolver) DataSourceSpecForSettlementData(ctx context.Context, obj *vega.Perpetual) (*DataSourceSpec, error) {
	var status DataSourceSpecStatus

	switch obj.DataSourceSpecForSettlementData.Status {
	case vega.DataSourceSpec_STATUS_ACTIVE:
		status = DataSourceSpecStatusStatusActive
	case vega.DataSourceSpec_STATUS_DEACTIVATED:
		status = DataSourceSpecStatusStatusDeactivated
	default:
		return nil, fmt.Errorf("unknown status: %v", obj.DataSourceSpecForSettlementData.Status)
	}

	return &DataSourceSpec{
		ID:        obj.DataSourceSpecForSettlementData.Id,
		CreatedAt: obj.DataSourceSpecForSettlementData.CreatedAt,
		UpdatedAt: &obj.DataSourceSpecForSettlementData.UpdatedAt,
		Data:      obj.DataSourceSpecForSettlementData.Data,
		Status:    status,
	}, nil
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
