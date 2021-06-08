package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/events"
	pb "code.vegaprotocol.io/vega/proto"
	apipb "code.vegaprotocol.io/vega/proto/api"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

func TestMarkets_GetAll(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		market := be.GetMarket()
		require.NotNil(t, market)
		e := events.NewMarketCreatedEvent(ctx, pb.Market{
			Id: market.MarketId,
		})
		return e, nil
	}, "markets-events.golden")

	// we also send a NewMarketUpdatedEvent for a market creation
	// See execution.Engine#publishMarketInfos
	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		market := be.GetMarket()
		e := events.NewMarketUpdatedEvent(ctx, pb.Market{
			Id: market.MarketId,
		})
		return e, nil
	}, "markets-events.golden")

	<-time.After(200 * time.Millisecond)

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	marketID := "a6f2c001f855f926b49bd43add22bc8bf619d569c3ef6fe442a3c31ffdc54fa5"

	resp, err := client.Markets(ctx, &apipb.MarketsRequest{})

	assert.NoError(t, err)
	assert.Len(t, resp.Markets, 1)
	assert.Equal(t, marketID, resp.Markets[0].Id)
}
