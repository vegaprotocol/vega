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
	"code.vegaprotocol.io/vega/types"
)

const defaultTimout = 2 * time.Second

func TestGetPartyAccounts(t *testing.T) {
	// setting up badger for parallel use seems to be causing problems
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		acc := be.GetAccount()
		e := events.NewAccountEvent(ctx, types.Account{
			Id:       acc.Id,
			Owner:    acc.Owner,
			Balance:  acc.Balance,
			Asset:    acc.Asset,
			MarketId: acc.MarketId,
			Type:     acc.Type,
		})
		return e, nil
	})

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	partyID := "6fb72005cde8e239f8d3b08c5fbcec06f93bfb45e9013208f662954923343fba"

	resp, err := client.PartyAccounts(ctx, &apipb.PartyAccountsRequest{
		PartyId: partyID,
		Type:    pb.AccountType_ACCOUNT_TYPE_GENERAL,
	})

	assert.NoError(t, err)
	assert.Len(t, resp.Accounts, 1)
	assert.Equal(t, partyID, resp.Accounts[0].Owner)
}
