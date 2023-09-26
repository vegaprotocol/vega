package gql

import (
	"context"
	"errors"
	"math"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/protos/vega"
)

type referralProgramResolver VegaResolverRoot

func (r *referralProgramResolver) Version(ctx context.Context, obj *vega.ReferralProgram) (int, error) {
	if obj.Version > math.MaxInt {
		return 0, errors.New("version is too large")
	}

	return int(obj.Version), nil
}

func (r *referralProgramResolver) WindowLength(ctx context.Context, obj *vega.ReferralProgram) (int, error) {
	if obj.WindowLength > math.MaxInt {
		return 0, errors.New("window length is too large")
	}

	return int(obj.WindowLength), nil
}

func (r *referralProgramResolver) EndOfProgramTimestamp(ctx context.Context, obj *vega.ReferralProgram) (string, error) {
	endTime := time.Unix(obj.EndOfProgramTimestamp, 0)
	return endTime.Format(time.RFC3339), nil
}

type currentReferralProgramResolver VegaResolverRoot

func (r *currentReferralProgramResolver) Version(ctx context.Context, obj *v2.ReferralProgram) (int, error) {
	if obj.Version > math.MaxInt {
		return 0, errors.New("version is too large")
	}

	return int(obj.Version), nil
}

func (r *currentReferralProgramResolver) WindowLength(ctx context.Context, obj *v2.ReferralProgram) (int, error) {
	if obj.WindowLength > math.MaxInt {
		return 0, errors.New("window length is too large")
	}

	return int(obj.WindowLength), nil
}
