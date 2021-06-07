package execution_test

import (
	"context"
	"testing"
	"time"

	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderBookAmends_FlipToGTT(t *testing.T) {
	now := time.Unix(5, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()
	defer tm.ctrl.Finish()

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTC, "Order01", types.Side_SIDE_BUY, "aaa", 2, 100)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)
	require.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())

	// now we edit the order t make it GTC so it should not expire
	amendment := &commandspb.OrderAmendment{
		OrderId:     o1.Id,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		ExpiresAt: &types.Timestamp{
			Value: now.Add(10 * time.Second).UnixNano(),
		},
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, "aaa")
	require.NotNil(t, amendConf)
	require.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_ACTIVE, amendConf.Order.Status)
	require.Equal(t, 1, tm.market.GetPeggedExpiryOrderCount())

	// now we edit the order t make it GTC so it should not expire
	amendment2 := &commandspb.OrderAmendment{
		OrderId:     o1.Id,
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
		ExpiresAt: &types.Timestamp{
			Value: now.Add(20 * time.Second).UnixNano(),
		},
	}

	amendConf2, err := tm.market.AmendOrder(ctx, amendment2, "aaa")
	require.NotNil(t, amendConf2)
	require.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_ACTIVE, amendConf2.Order.Status)
	require.Equal(t, 1, tm.market.GetPeggedExpiryOrderCount())

	// now we edit the order t make it GTC so it should not expire
	amendment3 := &commandspb.OrderAmendment{
		OrderId:     o1.Id,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}

	amendConf3, err := tm.market.AmendOrder(ctx, amendment3, "aaa")
	require.NotNil(t, amendConf3)
	require.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_ACTIVE, amendConf3.Order.Status)
	require.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())
}
