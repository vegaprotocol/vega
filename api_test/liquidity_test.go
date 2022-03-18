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
	"code.vegaprotocol.io/vega/types/num"
)

func TestLiquidity_Get(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		lp := be.GetLiquidityProvision()
		require.NotNil(t, lp)
		fee, _ := num.DecimalFromString(lp.Fee)

		var sells []*types.LiquidityOrderReference
		for _, v := range lp.Sells {
			s, err := types.LiquidityOrderReferenceFromProto(v)
			require.NoError(t, err)
			sells = append(sells, s)
		}

		var buys []*types.LiquidityOrderReference
		for _, v := range lp.Buys {
			b, err := types.LiquidityOrderReferenceFromProto(v)
			require.NoError(t, err)
			sells = append(buys, b)
		}
		commitmentAmount, _ := num.UintFromString(lp.CommitmentAmount, 10)
		e := events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
			ID:               lp.Id,
			Party:            lp.PartyId,
			CreatedAt:        lp.CreatedAt,
			UpdatedAt:        lp.UpdatedAt,
			MarketID:         lp.MarketId,
			CommitmentAmount: commitmentAmount,
			Fee:              fee,
			Sells:            sells,
			Buys:             buys,
			Version:          lp.Version,
			Status:           lp.Status,
			Reference:        lp.Reference,
		})
		return e, nil
	}, "liquidity-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	lpMmarketID := "076BB86A5AA41E3E"
	lpPartyID := "0f3d86044f8e7efff27131227235fb6db82574e24f788c30723d67f888b51d61"

	var respWithParty *apipb.LiquidityProvisionsResponse
	var respNoParty *apipb.LiquidityProvisionsResponse

	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			respNoParty, err = client.LiquidityProvisions(ctx, &apipb.LiquidityProvisionsRequest{
				Market: lpMmarketID,
			})
			require.NotNil(t, respNoParty)
			require.NoError(t, err)

			respWithParty, err = client.LiquidityProvisions(ctx, &apipb.LiquidityProvisionsRequest{
				Market: lpMmarketID,
				Party:  lpPartyID,
			})
			require.NotNil(t, respWithParty)
			require.NoError(t, err)

			if len(respWithParty.LiquidityProvisions) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)

	require.NotEmpty(t, respNoParty.LiquidityProvisions)
	require.NotEqual(t, "", respNoParty.String())

	assert.Equal(t, lpMmarketID, respWithParty.LiquidityProvisions[0].MarketId)
	assert.Equal(t, lpPartyID, respWithParty.LiquidityProvisions[0].PartyId)
}
