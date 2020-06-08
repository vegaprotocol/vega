package gql

import (
	"testing"
	"time"

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
	assert.Equal(t, types.Order_STATUS_ACTIVE, status)
	expired := OrderStatusExpired
	status, err = parseOrderStatus(&expired)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_STATUS_EXPIRED, status)
	cancelled := OrderStatusCancelled
	status, err = parseOrderStatus(&cancelled)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_STATUS_CANCELLED, status)
	unknown := OrderStatus("好候")
	_, err = parseOrderStatus(&unknown)
	assert.Error(t, err)
}

func TestParseOrderTimeInForce(t *testing.T) {
	fok := OrderTimeInForceFok
	orderType, err := parseOrderTimeInForce(fok)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_TIF_FOK, orderType)
	ioc := OrderTimeInForceIoc
	orderType, err = parseOrderTimeInForce(ioc)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_TIF_IOC, orderType)
	gtt := OrderTimeInForceGtt
	orderType, err = parseOrderTimeInForce(gtt)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_TIF_GTT, orderType)
	gtc := OrderTimeInForceGtc
	orderType, err = parseOrderTimeInForce(gtc)
	assert.Nil(t, err)
	assert.Equal(t, types.Order_TIF_GTC, orderType)
	unknown := OrderTimeInForce("好到时候")
	_, err = parseOrderTimeInForce(unknown)
	assert.Error(t, err)

}

func TestParseSide(t *testing.T) {
	buy := SideBuy
	side, err := parseSide(&buy)
	assert.Nil(t, err)
	assert.Equal(t, types.Side_SIDE_BUY, side)
	sell := SideSell
	side, err = parseSide(&sell)
	assert.Nil(t, err)
	assert.Equal(t, types.Side_SIDE_SELL, side)
	unknown := Side("好到时候")
	_, err = parseSide(&unknown)
	assert.Error(t, err)
}

func TestSecondsTSToDatetime(t *testing.T) {
	aTime := "2020-05-30T00:00:00Z"
	testTime, err := time.Parse(time.RFC3339Nano, aTime)
	assert.NoError(t, err)

	stringified := secondsTSToDatetime(testTime.Unix())
	assert.EqualValues(t, aTime, stringified)

	badValue := secondsTSToDatetime(testTime.UnixNano())
	assert.NotEqual(t, aTime, badValue)
}

func TestNanoTSToDatetime(t *testing.T) {
	aTime := "2020-05-30T00:00:00Z"
	testTime, err := time.Parse(time.RFC3339Nano, aTime)
	assert.NoError(t, err)

	stringified := nanoTSToDatetime(testTime.UnixNano())
	assert.EqualValues(t, aTime, stringified)

	badValue := nanoTSToDatetime(testTime.Unix())
	assert.NotEqual(t, aTime, badValue)
}
