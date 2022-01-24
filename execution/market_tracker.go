package execution

import "code.vegaprotocol.io/vega/types/num"

type MarketTracker struct {
	marketID      string
	volumeTraded  *num.Uint
	proposersPaid bool
}
