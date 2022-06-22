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
	pb "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

func TestPartyByID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		party := be.GetParty()
		require.NotNil(t, party)
		e := events.NewPartyEvent(ctx, pb.Party{
			Id: party.Id,
		})
		return e, nil
	}, "parties-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	partyID := "c1f55d6be5dddbbff20312e1103a6f4b86ff4a798b74d7e9c980f98fb6747c11"

	var resp *apipb.PartyByIDResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.PartyByID(ctx, &apipb.PartyByIDRequest{
				PartyId: partyID,
			})
			if err == nil && resp != nil && resp.Party != nil {
				break loop
			}
		}
	}

	require.NotNil(t, t, resp.Party)
	assert.Equal(t, partyID, resp.Party.Id)
}
