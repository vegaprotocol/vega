package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/events"
	apipb "code.vegaprotocol.io/vega/proto/api"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
)

func TestGetSpecs(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
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

	<-time.After(200 * time.Millisecond)

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	oracleSpecID := "6f9b102855efc7b2421df3de4007bd3c6b9fd237e0f9b9b18326800fd822184f"

	resp, err := client.OracleSpecs(ctx, &apipb.OracleSpecsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, oracleSpecID, resp.OracleSpecs[0].Id)
}
