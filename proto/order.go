package proto

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
