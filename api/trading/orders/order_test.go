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

func TestOrder_JsonReturnsValidJson(t *testing.T) {
	var o models.Order

	o = buildNewOrder()

	jsonO, _ := o.Json()

	assert.Equal(
		t,
		"{\"id\":\"d41d8cd98f00b204e9800998ecf8427e\",\"market\":\"market\",\"party\":\"party\",\"side\":0,\"price\":1,\"size\":1,\"remaining\":1,\"timestamp\":1234567890,\"type\":1}",
		string(jsonO),
	)

}

func TestOrder_JsonWithEncodingReturnsValidAndEncodedJson(t *testing.T) {
	var o models.Order

	o = buildNewOrder()

	jsonEnc, _ := o.JsonWithEncoding()

	assert.Equal(
		t,
		"eyJpZCI6ImQ0MWQ4Y2Q5OGYwMGIyMDRlOTgwMDk5OGVjZjg0MjdlIiwibWFya2V0IjoibWFya2V0IiwicGFydHkiOiJwYXJ0eSIsInNpZGUiOjAsInByaWNlIjoxLCJzaXplIjoxLCJyZW1haW5pbmciOjEsInRpbWVzdGFtcCI6MTIzNDU2Nzg5MCwidHlwZSI6MX0=",
		string(jsonEnc),
	)
}

// Helpers
func buildNewOrder() models.Order  {
 	return models.NewOrder("d41d8cd98f00b204e9800998ecf8427e","market", "party", 0, 1,1, 1, 1234567890, 1)
}
