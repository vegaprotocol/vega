package matching

import "code.vegaprotocol.io/vega/core/types"

func (b *OrderBook) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	o, err := b.RemoveOrderWithStatus(order.ID, types.OrderStatusCancelled)
	if err != nil {
		return nil, err
	}
	return &types.OrderCancellationConfirmation{o}, err
}
