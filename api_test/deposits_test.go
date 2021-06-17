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

func TestDeposits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		deposit := be.GetDeposit()
		require.NotNil(t, deposit)
		e := events.NewDepositEvent(ctx, pb.Deposit{
			Id:                deposit.Id,
			Status:            deposit.Status,
			PartyId:           deposit.PartyId,
			Asset:             deposit.Asset,
			Amount:            deposit.Amount,
			TxHash:            deposit.TxHash,
			CreditedTimestamp: deposit.CreditedTimestamp,
			CreatedTimestamp:  deposit.CreatedTimestamp,
		})
		return e, nil
	}, "deposit-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	depositID := "af6e66ee1e1a643338f55b8dfe00129b09b926a997edddf1f10e76b31c65cdad"
	depositPartyID := "c5fdc709b3464ca10292437ce493dc0e497b2c3ea22a5fde714c4e487b93011d"

	var resp *apipb.DepositsResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(1 * time.Millisecond):
			resp, err = client.Deposits(ctx, &apipb.DepositsRequest{
				PartyId: depositPartyID,
			})
			require.NotNil(t, resp)
			require.NoError(t, err)
			if len(resp.Deposits) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Equal(t, depositID, resp.Deposits[0].Id)
	assert.Equal(t, depositPartyID, resp.Deposits[0].PartyId)
}
