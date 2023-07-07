package gql

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type orderSubmissionResolver VegaResolverRoot

func (r orderSubmissionResolver) Size(_ context.Context, obj *commandspb.OrderSubmission) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}

func (r orderSubmissionResolver) IcebergOrder(_ context.Context, obj *commandspb.OrderSubmission) (*vega.IcebergOrder, error) {
	if obj.IcebergOpts == nil {
		return nil, nil
	}

	return &vega.IcebergOrder{
		PeakSize:           obj.IcebergOpts.PeakSize,
		MinimumVisibleSize: obj.IcebergOpts.MinimumVisibleSize,
	}, nil
}
