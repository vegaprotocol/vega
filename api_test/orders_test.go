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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
)

func TestGetByOrderID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		order, err := types.OrderFromProto(be.GetOrder())
		require.NotNil(t, order)
		require.NoError(t, err)
		e := events.NewOrderEvent(ctx, order)
		return e, nil
	}, "orders-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
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
