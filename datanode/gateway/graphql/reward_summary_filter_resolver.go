package gql

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type rewardSummaryFilterResolver VegaResolverRoot

func (r *rewardSummaryFilterResolver) FromEpoch(_ context.Context, obj *v2.RewardSummaryFilter, data *int) (err error) {
	obj.FromEpoch, err = intPtrToUint64Ptr(data)
	return
}

func (r *rewardSummaryFilterResolver) ToEpoch(_ context.Context, obj *v2.RewardSummaryFilter, data *int) (err error) {
	obj.ToEpoch, err = intPtrToUint64Ptr(data)
	return
}

func intPtrToUint64Ptr(i *int) (*uint64, error) {
	iVal := ptr.UnBox(i)
	if iVal < 0 {
		return nil, fmt.Errorf("cannot convert to uint - must be positive")
	}
	return ptr.From(uint64(iVal)), nil
}
