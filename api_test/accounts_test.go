package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/types"
	"code.vegaprotocol.io/data-node/types/num"
	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	pb "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

func TestGetPartyAccounts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		acc := be.GetAccount()
		require.NotNil(t, acc)
		e := events.NewAccountEvent(ctx, types.Account{
			Id:       acc.Id,
			Owner:    acc.Owner,
			Balance:  num.NewUint(acc.Balance),
			Asset:    acc.Asset,
			MarketId: acc.MarketId,
			Type:     acc.Type,
		})
		return e, nil
	}, "account-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	partyID := "6fb72005cde8e239f8d3b08c5fbcec06f93bfb45e9013208f662954923343fba"

	var resp *apipb.PartyAccountsResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.PartyAccounts(ctx, &apipb.PartyAccountsRequest{
				PartyId: partyID,
				Type:    pb.AccountType_ACCOUNT_TYPE_GENERAL,
			})
			if err == nil && len(resp.Accounts) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Len(t, resp.Accounts, 1)
	assert.Equal(t, partyID, resp.Accounts[0].Owner)
}
