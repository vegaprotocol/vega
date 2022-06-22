// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package api_test

import (
	"context"
	"testing"
	"time"

	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByMarket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		trade := be.GetTrade()
		require.NotNil(t, trade)
		e := events.NewTradeEvent(ctx, *TradeFromProto(trade))
		return e, nil
	}, "trades-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
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
