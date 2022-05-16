package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
)

func TestOracleSpecs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		spec := be.GetOracleSpec()
		require.NotNil(t, spec)
		e := events.NewOracleSpecEvent(ctx, oraclespb.OracleSpec{
			Id:        spec.Id,
			CreatedAt: spec.CreatedAt,
			UpdatedAt: spec.UpdatedAt,
			PubKeys:   spec.PubKeys,
			Filters:   spec.Filters,
			Status:    spec.Status,
		})
		return e, nil
	}, "oracle-spec-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	oracleSpecID := "6f9b102855efc7b2421df3de4007bd3c6b9fd237e0f9b9b18326800fd822184f"

	var resp *apipb.OracleSpecsResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.OracleSpecs(ctx, &apipb.OracleSpecsRequest{})
			require.NotNil(t, resp)
			require.NoError(t, err)
			if len(resp.OracleSpecs) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Equal(t, oracleSpecID, resp.OracleSpecs[0].Id)
}
