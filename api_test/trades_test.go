package api_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/events"
	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByMarket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		trade := be.GetTrade()
		require.NotNil(t, trade)
		e := events.NewTradeEvent(ctx, *TradeFromProto(trade))
		return e, nil
	}, "trades-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	tradeID := "V0000030271-0001798304-0000000000"
	tradeMarketID := "2839D9B2329C9E70"

	var resp *apipb.TradesByMarketResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.TradesByMarket(ctx, &apipb.TradesByMarketRequest{
				MarketId:   tradeMarketID,
				Pagination: nil,
			})
			if err == nil && len(resp.Trades) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Equal(t, tradeID, resp.Trades[0].Id)
	assert.Equal(t, tradeMarketID, resp.Trades[0].MarketId)
}
