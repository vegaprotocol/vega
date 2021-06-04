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

func TestLiquidity_Get(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		lp := be.GetLiquidityProvision()
		require.NotNil(t, lp)
		e := events.NewLiquidityProvisionEvent(ctx, &pb.LiquidityProvision{
			Id:               lp.Id,
			PartyId:          lp.PartyId,
			CreatedAt:        lp.CreatedAt,
			UpdatedAt:        lp.UpdatedAt,
			MarketId:         lp.MarketId,
			CommitmentAmount: lp.CommitmentAmount,
			Fee:              lp.Fee,
			Sells:            lp.Sells,
			Buys:             lp.Buys,
			Version:          lp.Version,
			Status:           lp.Status,
			Reference:        lp.Reference,
		})
		return e, nil
	}, "liquidity-events.golden")

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	lpMmarketID := "076BB86A5AA41E3E"
	lpPartyID := "0f3d86044f8e7efff27131227235fb6db82574e24f788c30723d67f888b51d61"

	resp, err := client.LiquidityProvisions(ctx, &apipb.LiquidityProvisionsRequest{
		Market: lpMmarketID,
		Party:  lpPartyID,
	})

	assert.NoError(t, err)
	assert.Equal(t, lpMmarketID, resp.LiquidityProvisions[0].MarketId)
	assert.Equal(t, lpPartyID, resp.LiquidityProvisions[0].PartyId)
}
