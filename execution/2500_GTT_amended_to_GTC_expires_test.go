package execution_test

import (
	"context"
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGTTAmendToGTCAmendInPlace_OrderGetExpired(t *testing.T) {
	now := time.Unix(5, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.Order_TYPE_LIMIT, types.Order_TIME_IN_FORCE_GTT, "Order01", types.Side_SIDE_BUY, "aaa", 1, 10)
	o1.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	// now we edit the order t make it GTC so it should not expire
	amendment := &commandspb.OrderAmendment{
		OrderId:     o1.Id,
		PartyId:     "aaa",
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment)
	require.NotNil(t, amendConf)
	require.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_ACTIVE, amendConf.Order.Status)

	// now expire, and nothing should be returned
	tm.market.OnChainTimeUpdate(context.Background(), now.Add(10*time.Second))
	orders, err := tm.market.RemoveExpiredOrders(
		context.Background(), now.UnixNano())
	require.Equal(t, 0, len(orders))
	require.NoError(t, err)
}
