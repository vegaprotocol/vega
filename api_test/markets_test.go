package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/data-node/events"
	apipb "code.vegaprotocol.io/data-node/proto/api"
	eventspb "code.vegaprotocol.io/data-node/proto/vega/events/v1"
	"code.vegaprotocol.io/data-node/types"
)

func TestMarkets_GetAll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		market := be.GetMarket()
		require.NotNil(t, market)
		e := events.NewMarketCreatedEvent(ctx, types.Market{
			Id: market.MarketId,
		})
		return e, nil
	}, "markets-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	marketID := "a6f2c001f855f926b49bd43add22bc8bf619d569c3ef6fe442a3c31ffdc54fa5"

	var resp *apipb.MarketsResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.Markets(ctx, &apipb.MarketsRequest{})
			require.NotNil(t, resp)
			require.NoError(t, err)
			if len(resp.Markets) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Len(t, resp.Markets, 1)
	assert.Equal(t, marketID, resp.Markets[0].Id)
}
