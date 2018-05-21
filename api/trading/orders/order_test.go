package orders

import (
	"testing"
	"github.com/magiconair/properties/assert"
)

func TestNewOrder(t *testing.T) {
	var o Order

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
	var o Order

	o = buildNewOrder()

	jsonO, _ := o.Json()

	assert.Equal(
		t,
		string(jsonO),
		"{\"Market\":\"market\",\"Party\":\"party\",\"Side\":0,\"Price\":1,\"Size\":1,\"Remaining\":1,\"Timestamp\":1234567890,\"Type\":1}",
	)

}

func TestOrder_JsonWithEncoding_ReturnsValidAndEncodedJson(t *testing.T) {
	var o Order

	o = buildNewOrder()

	jsonEnc, _ := o.JsonWithEncoding()

	assert.Equal(
		t,
		string(jsonEnc),
		"eyJNYXJrZXQiOiJtYXJrZXQiLCJQYXJ0eSI6InBhcnR5IiwiU2lkZSI6MCwiUHJpY2UiOjEsIlNpemUiOjEsIlJlbWFpbmluZyI6MSwiVGltZXN0YW1wIjoxMjM0NTY3ODkwLCJUeXBlIjoxfQ==",
	)
}

// Helpers
func buildNewOrder() Order  {
 	return NewOrder("market", "party", 0, 1,1, 1, 1234567890, 1)
}
