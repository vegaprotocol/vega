package gql

import (
	"context"
	"errors"
	"math"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type volumeDiscountStatsResolver VegaResolverRoot

func (v *volumeDiscountStatsResolver) AtEpoch(_ context.Context, obj *v2.VolumeDiscountStats) (int, error) {
	if obj.AtEpoch > math.MaxInt {
		return 0, errors.New("at_epoch is too large")
	}

	return int(obj.AtEpoch), nil
}
