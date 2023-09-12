package gql

import (
	"context"
	"errors"
	"math"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type referralProgramResolver VegaResolverRoot

func (r *referralProgramResolver) Version(ctx context.Context, obj *v2.ReferralProgram) (int, error) {
	if obj.Version > math.MaxInt {
		return 0, errors.New("version is too large")
	}

	return int(obj.Version), nil
}

func (r *referralProgramResolver) WindowLength(ctx context.Context, obj *v2.ReferralProgram) (int, error) {
	if obj.WindowLength > math.MaxInt {
		return 0, errors.New("window length is too large")
	}

	return int(obj.WindowLength), nil
}
