package orders

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"vega/api/trading/orders/models"
)

func TestNewOrder(t *testing.T) {
	var o models.Order

	o = buildNewOrder()

	assert.Equal(t, o.Market, "market")
	assert.Equal(t, o.Party, "party")
	assert.Equal(t, o.Side, int32(0))
	assert.Equal(t, o.Price, uint64(1))
	assert.Equal(t, o.Remaining, uint64(1))
	assert.Equal(t, o.Timestamp, uint64(1234567890))
	assert.Equal(t, o.Type, 1)
	assert.Equal(t, o.Size, uint64(1))
}

func TestOrder_Json_ReturnsValidJson(t *testing.T) {
	var o models.Order

	o = buildNewOrder()

	jsonO, _ := o.Json()

	assert.Equal(
		t,
		"{\"market\":\"market\",\"party\":\"party\",\"side\":0,\"price\":1,\"size\":1,\"remaining\":1,\"timestamp\":1234567890,\"type\":1}",
		string(jsonO),
	)

}

func TestOrder_JsonWithEncoding_ReturnsValidAndEncodedJson(t *testing.T) {
	var o models.Order

	o = buildNewOrder()

	jsonEnc, _ := o.JsonWithEncoding()

	assert.Equal(
		t,
		"eyJtYXJrZXQiOiJtYXJrZXQiLCJwYXJ0eSI6InBhcnR5Iiwic2lkZSI6MCwicHJpY2UiOjEsInNpemUiOjEsInJlbWFpbmluZyI6MSwidGltZXN0YW1wIjoxMjM0NTY3ODkwLCJ0eXBlIjoxfQ==",
		string(jsonEnc),
	)
}

// Helpers
func buildNewOrder() models.Order  {
 	return models.NewOrder("market", "party", 0, 1,1, 1, 1234567890, 1)
}
