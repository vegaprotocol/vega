package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/events"
	apipb "code.vegaprotocol.io/vega/proto/api"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/types"
)

func TestGetByOrderID(t *testing.T) {
	t.Parallel()
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
		case <-time.Tick(1 * time.Millisecond):
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
