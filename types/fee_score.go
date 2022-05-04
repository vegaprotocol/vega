package types

import (
	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
)

// FeePartyScore represents the fraction the party has in the total fee.
type PartyContibutionScore struct {
	Party string
	Score num.Decimal
}

type MarketContributionScore struct {
	Asset  string
	Market string
	Metric proto.DispatchMetric
	Score  num.Decimal
}
