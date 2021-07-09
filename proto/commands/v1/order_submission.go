package v1

import (
	types "code.vegaprotocol.io/vega/proto"
)

func (s *OrderSubmission) IntoOrder(party string) *types.Order {
	return &types.Order{
		MarketId:    s.MarketId,
		PartyId:     party,
		Price:       s.Price,
		Size:        s.Size,
		Side:        s.Side,
		TimeInForce: s.TimeInForce,
		Type:        s.Type,
		ExpiresAt:   s.ExpiresAt,
		Reference:   s.Reference,
		Status:      types.Order_STATUS_ACTIVE,
		Remaining:   s.Size,
		PeggedOrder: s.PeggedOrder,
	}
}
