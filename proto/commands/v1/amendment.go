package v1

import (
	types "code.vegaprotocol.io/vega/proto"

	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
)

func AmendSize(o *types.Order, newSize int64) *OrderAmendment {
	a := &OrderAmendment{
		OrderId:  o.Id,
		MarketId: o.MarketId,

		SizeDelta:   newSize - int64(o.Size),
		TimeInForce: o.TimeInForce,
	}
	if e := o.ExpiresAt; e > 0 {
		a.ExpiresAt = &types.Timestamp{
			Value: e,
		}
	}

	if p := o.PeggedOrder; p != nil {
		a.PeggedReference = p.Reference
		a.PeggedOffset = &wrapperspb.Int64Value{
			Value: p.Offset,
		}
	} else {
		if p := o.Price; p > 0 {
			a.Price = &types.Price{
				Value: p,
			}
		}
	}

	return a
}
