package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/types"
	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

func TestGetByOrderID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		order := be.GetOrder()
		require.NotNil(t, order)
		e := events.NewOrderEvent(ctx, types.OrderFromProto(order))
		return e, nil
	}, "orders-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	orderID := "V0000000567-0000005166"

	var resp *apipb.OrderByIDResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.OrderByID(ctx, &apipb.OrderByIDRequest{
				OrderId: orderID,
				Version: 1,
			})
			if err == nil && resp.Order != nil {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Equal(t, orderID, resp.Order.Id)
}
