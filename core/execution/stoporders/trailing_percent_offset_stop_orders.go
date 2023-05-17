package stoporders

import (
	"code.vegaprotocol.io/vega/core/types"
)

type TrailingPercentOffsetStopOrders struct {
}

func (p *TrailingPercentOffsetStopOrders) Insert(order *types.StopOrder) {}
func (p *TrailingPercentOffsetStopOrders) Remove(id string)              {}
