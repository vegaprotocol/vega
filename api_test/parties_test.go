package api_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/events"
	pb "code.vegaprotocol.io/vega/proto"
	apipb "code.vegaprotocol.io/vega/proto/api"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

func TestParties(t *testing.T) {
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

	<-time.After(200 * time.Millisecond)

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	partyID := "c1f55d6be5dddbbff20312e1103a6f4b86ff4a798b74d7e9c980f98fb6747c11"

	resp, err := client.Parties(ctx, &apipb.PartiesRequest{})
	require.NotNil(t, resp)
	require.NoError(t, err)

	sortedParties := resp.Parties
	sort.Slice(sortedParties, func(i, j int) bool {
		return sortedParties[i].Id > sortedParties[j].Id
	})

	assert.Equal(t, "network", sortedParties[0].Id)
	assert.Equal(t, partyID, sortedParties[1].Id)
}
