package types

import "code.vegaprotocol.io/vega/types/num"

// FeePartyScore represents the fraction the party has in the total fee.
type FeePartyScore struct {
	Party string
	Score num.Decimal
}
