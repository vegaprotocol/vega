package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/events"
	pb "code.vegaprotocol.io/vega/proto"
	apipb "code.vegaprotocol.io/vega/proto/api"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

func TestDeposits(t *testing.T) {
	t.Parallel()
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
	}, "oracle-spec-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	depositID := "6f9b102855efc7b2421df3de4007bd3c6b9fd237e0f9b9b18326800fd822184f"

	resp, err := client.Deposits(ctx, &apipb.DepositsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, depositID, resp.Deposits[0].Id)
}
