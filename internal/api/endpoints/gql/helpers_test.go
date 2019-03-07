package gql

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestSafeStringUint64(t *testing.T) {
	var convTests = []struct {
		in          string
		out         uint64
		expectError bool
	}{
		{"-1", 0, true},
		{"-9223372036854775808", 0, true},
		{"x';INSERT INTO users ('email','passwd') VALUES ('ned@fladers.org','hello');--", 0, true},
		{"0", 0, false},
		{"100", 100, false},
		{"9223372036854775807", 9223372036854775807, false},
		{"18446744073709551615", 18446744073709551615, false},
	}

	for _, tt := range convTests {

		c, err := safeStringUint64(tt.in)

		assert.Equal(t, tt.out, c)

		if tt.expectError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestParseOrderStatus(t *testing.T) {
	active := OrderStatusActive
	status, err := parseOrderStatus(&active)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_Active, status)
	expired := OrderStatusExpired
	status, err = parseOrderStatus(&expired)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_Expired, status)
	cancelled := OrderStatusCancelled
	status, err = parseOrderStatus(&cancelled)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_Cancelled, status)
	unknown := OrderStatus("好候")
	status, err = parseOrderStatus(&unknown)
	assert.Error(t, err)
}

func TestParseOrderType(t *testing.T) {
	fok := OrderTypeFok
	orderType, err := parseOrderType(&fok)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_FOK, orderType)
	ene := OrderTypeEne
	orderType, err = parseOrderType(&ene)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_ENE, orderType)
	gtt := OrderTypeGtt
	orderType, err = parseOrderType(&gtt)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_GTT, orderType)
	gtc := OrderTypeGtc
	orderType, err = parseOrderType(&gtc)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_GTC, orderType)
	unknown := OrderType("好到时候")
	orderType, err = parseOrderType(&unknown)
	assert.Error(t, err)

}

func TestParseSide(t *testing.T) {
	buy := SideBuy
	side, err := parseSide(&buy)
	assert.Nil(t, err)
	assert.Equal(t, types.Side_Buy, side)
	sell := SideSell
	side, err = parseSide(&sell)
	assert.Nil(t, err)
	assert.Equal(t, types.Side_Sell, side)
	unknown := Side("好到时候")
	side, err = parseSide(&unknown)
	assert.Error(t, err)
}
