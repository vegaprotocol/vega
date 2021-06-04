package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/events"
	pb "code.vegaprotocol.io/vega/proto"
	apipb "code.vegaprotocol.io/vega/proto/api"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

func TestGetByOrderID(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		order := be.GetOrder()
		require.NotNil(t, order)
		e := events.NewOrderEvent(ctx, &pb.Order{
			Id:                   order.Id,
			MarketId:             order.MarketId,
			PartyId:              order.PartyId,
			Side:                 order.Side,
			Price:                order.Price,
			Size:                 order.Size,
			Remaining:            order.Remaining,
			TimeInForce:          order.TimeInForce,
			Type:                 order.Type,
			CreatedAt:            order.CreatedAt,
			Status:               order.Status,
			ExpiresAt:            order.ExpiresAt,
			Reference:            order.Reference,
			Reason:               order.Reason,
			UpdatedAt:            order.UpdatedAt,
			Version:              order.Version,
			BatchId:              order.BatchId,
			PeggedOrder:          order.PeggedOrder,
			LiquidityProvisionId: order.LiquidityProvisionId,
		})
		return e, nil
	}, "orders-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	orderID := "V0000000567-0000005166"

	resp, err := client.OrderByID(ctx, &apipb.OrderByIDRequest{
		OrderId: orderID,
		Version: 1,
	})

	assert.NoError(t, err)
	assert.Equal(t, orderID, resp.Order.Id)
}
