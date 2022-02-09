package execution_test

import (
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"context"
	"encoding/hex"
	"math/rand"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGTTAmendToGTCAmendInPlace_OrderGetExpired(t *testing.T) {
	now := time.Unix(5, 0)
	closingAt := time.Unix(10000000000, 0)
	tm := getTestMarket(t, now, closingAt, nil, nil)
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, randomSha256Hash())

	addAccount(tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order01", types.SideBuy, "aaa", 1, 10)
	o1.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	// now we edit the order t make it GTC so it should not expire
	amendment := &types.OrderAmendment{
		OrderID:     o1.ID,
		TimeInForce: types.OrderTimeInForceGTC,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, "aaa", randomSha256Hash())
	require.NotNil(t, amendConf)
	require.NoError(t, err)
	assert.Equal(t, types.OrderStatusActive, amendConf.Order.Status)

	// now expire, and nothing should be returned
	ctx = vegacontext.WithTraceID(context.Background(), randomSha256Hash())
	tm.market.OnChainTimeUpdate(ctx, now.Add(10*time.Second))
	orders, err := tm.market.RemoveExpiredOrders(
		context.Background(), now.UnixNano())
	require.Equal(t, 0, len(orders))
	require.NoError(t, err)
}

func randomSha256Hash() string {
	data := make([]byte, 10)
	rand.Read(data)
	return hex.EncodeToString(crypto.Hash(data))
}
