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

func TestPartyByID(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		party := be.GetParty()
		require.NotNil(t, party)
		e := events.NewPartyEvent(ctx, pb.Party{
			Id: party.Id,
		})
		return e, nil
	}, "parties-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	partyID := "c1f55d6be5dddbbff20312e1103a6f4b86ff4a798b74d7e9c980f98fb6747c11"

	var resp *apipb.PartyByIDResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(1 * time.Millisecond):
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
