package gql

import (
	"context"
	"errors"
	"math"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type volumeDiscountProgramResolver VegaResolverRoot

func (r *volumeDiscountProgramResolver) Version(_ context.Context, obj *v2.VolumeDiscountProgram) (int, error) {
	if obj.Version > math.MaxInt {
		return 0, errors.New("version is too large")
	}

	return int(obj.Version), nil
}

func (r *volumeDiscountProgramResolver) WindowLength(_ context.Context, obj *v2.VolumeDiscountProgram) (int, error) {
	if obj.WindowLength > math.MaxInt {
		return 0, errors.New("window length is too large")
	}

	return int(obj.WindowLength), nil
}
