package gql

import (
	"context"

	vega "code.vegaprotocol.io/vega/protos/vega"
)

type updateVolumeDiscountProgramResolver VegaResolverRoot

func (r *updateVolumeDiscountProgramResolver) Version(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) (int, error) {
	return int(obj.Changes.Version), nil
}

func (r *updateVolumeDiscountProgramResolver) ID(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) (string, error) {
	return obj.Changes.Id, nil
}

func (r *updateVolumeDiscountProgramResolver) BenefitTiers(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) ([]*vega.VolumeBenefitTier, error) {
	return obj.Changes.BenefitTiers, nil
}

func (r *updateVolumeDiscountProgramResolver) EndOfProgramTimestamp(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) (int64, error) {
	return obj.Changes.EndOfProgramTimestamp, nil
}

func (r *updateVolumeDiscountProgramResolver) WindowLength(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) (int, error) {
	return int(obj.Changes.WindowLength), nil
}
