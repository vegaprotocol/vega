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

func TestWithdrawals(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		withdrawal := be.GetWithdrawal()
		require.NotNil(t, withdrawal)
		e := events.NewWithdrawalEvent(ctx, pb.Withdrawal{
			Id:                 withdrawal.Id,
			PartyId:            withdrawal.PartyId,
			Amount:             withdrawal.Amount,
			Asset:              withdrawal.Asset,
			Status:             withdrawal.Status,
			Ref:                withdrawal.Ref,
			Expiry:             withdrawal.Expiry,
			TxHash:             withdrawal.TxHash,
			CreatedTimestamp:   withdrawal.CreatedTimestamp,
			WithdrawnTimestamp: withdrawal.WithdrawnTimestamp,
		})
		return e, nil
	}, "withdrawals-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	withdrawalID := "6f9b102855efc7b2421df3de4007bd3c6b9fd237e0f9b9b18326800fd822184f"

	resp, err := client.Withdrawals(ctx, &apipb.WithdrawalsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, withdrawalID, resp.Withdrawals[0].Id)
}
