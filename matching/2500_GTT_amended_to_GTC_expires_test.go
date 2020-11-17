package matching

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestGTTAmendToGTCAmendInPlace_OrderGetExpired(t *testing.T) {
	market := "testmarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	originalOrder := types.Order{
		MarketID:    market,
		Status:      types.Order_STATUS_ACTIVE,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        4,
		Remaining:   4,
		TimeInForce: types.Order_TIF_GTT,
		ExpiresAt:   10,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000001",
	}
	amendedOrder := originalOrder
	amendedOrder.TimeInForce = types.Order_TIF_GTC
	amendedOrder.ExpiresAt = 0

	_, err := book.SubmitOrder(&originalOrder)
	assert.NoError(t, err)

	// now we send the amended order
	err = book.AmendOrder(&originalOrder, &amendedOrder)
	assert.NoError(t, err)

	// now we call the remove expires orders
	removedOrders := book.RemoveExpiredOrders(11)
	assert.Len(t, removedOrders, 0)
}
