package proto

import "google.golang.org/protobuf/types/known/wrapperspb"

// IsPersistent returns true if the order is persistent.
// A persistent order is a Limit type order that might be
// matched in the future.
func (o *Order) IsPersistent() bool {
	return (o.TimeInForce == Order_TIF_GTC ||
		o.TimeInForce == Order_TIF_GTT ||
		o.TimeInForce == Order_TIF_GFN ||
		o.TimeInForce == Order_TIF_GFA) &&
		o.Type == Order_TYPE_LIMIT &&
		o.Remaining > 0
}

func (o *Order) AmendSize(newSize int64) *OrderAmendment {
	a := &OrderAmendment{
		OrderID:  o.Id,
		MarketID: o.MarketID,
		PartyID:  o.PartyID,
		ExpiresAt: &Timestamp{
			Value: o.ExpiresAt,
		},
		SizeDelta:   newSize - int64(o.Size),
		TimeInForce: o.TimeInForce,
	}
	if p := o.Price; p > 0 {
		a.Price = &Price{
			Value: p,
		}
	}

	if p := o.PeggedOrder; p != nil {
		a.PeggedReference = p.Reference
		a.PeggedOffset = &wrapperspb.Int64Value{
			Value: p.Offset,
		}
	}

	return a
}
