package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type fundingPaymentResolver VegaResolverRoot

func (f *fundingPaymentResolver) FundingPeriodSeq(ctx context.Context, obj *v2.FundingPayment) (int, error) {
	return int(obj.FundingPeriodSeq), nil
}
