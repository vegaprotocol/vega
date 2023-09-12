package gql

import (
	"context"
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/protos/vega"
)

type benefitTierResolver VegaResolverRoot

func (br *benefitTierResolver) MinimumEpochs(_ context.Context, obj *vega.BenefitTier) (int, error) {
	minEpochs, err := strconv.ParseInt(obj.MinimumEpochs, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse minimum epochs %s: %v", obj.MinimumEpochs, err)
	}

	return int(minEpochs), nil
}
