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

func TestGetByMarket(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		trade := be.GetTrade()
		require.NotNil(t, trade)
		e := events.NewTradeEvent(ctx, *types.TradeFromProto(trade))
		return e, nil
	}, "trades-events.golden")

	<-time.After(200 * time.Millisecond)

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	tradeID := "V0000030271-0001798304-0000000000"
	tradeMarketID := "2839D9B2329C9E70"

	resp, err := client.TradesByMarket(ctx, &apipb.TradesByMarketRequest{
		MarketId:   tradeMarketID,
		Pagination: nil,
	})

	assert.NoError(t, err)
	assert.Equal(t, tradeID, resp.Trades[0].Id)
	assert.Equal(t, tradeMarketID, resp.Trades[0].MarketId)
}
