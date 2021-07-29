package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/data-node/events"
	apipb "code.vegaprotocol.io/protos/data-node/api"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/data-node/types"
	"code.vegaprotocol.io/data-node/types/num"
)

func TestLiquidity_Get(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		lp := be.GetLiquidityProvision()
		require.NotNil(t, lp)
		fee, _ := num.DecimalFromString(lp.Fee)

		var sells []*types.LiquidityOrderReference
		for _, v := range lp.Sells {
			s := types.LiquidityOrderReferenceFromProto(v)
			sells = append(sells, s)
		}

		var buys []*types.LiquidityOrderReference
		for _, v := range lp.Buys {
			b := types.LiquidityOrderReferenceFromProto(v)
			sells = append(buys, b)
		}

		e := events.NewLiquidityProvisionEvent(ctx, &types.LiquidityProvision{
			Id:               lp.Id,
			PartyId:          lp.PartyId,
			CreatedAt:        lp.CreatedAt,
			UpdatedAt:        lp.UpdatedAt,
			MarketId:         lp.MarketId,
			CommitmentAmount: num.NewUint(lp.CommitmentAmount),
			Fee:              fee,
			Sells:            sells,
			Buys:             buys,
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

	var resp *apipb.LiquidityProvisionsResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.LiquidityProvisions(ctx, &apipb.LiquidityProvisionsRequest{
				Market: lpMmarketID,
				Party:  lpPartyID,
			})
			require.NotNil(t, resp)
			require.NoError(t, err)
			if len(resp.LiquidityProvisions) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Equal(t, lpMmarketID, resp.LiquidityProvisions[0].MarketId)
	assert.Equal(t, lpPartyID, resp.LiquidityProvisions[0].PartyId)
}
