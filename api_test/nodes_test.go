package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	"code.vegaprotocol.io/vega/events"
)

func TestGetKeyRotations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	server.broker.Send(events.NewValidatorUpdateEvent(
		ctx,
		"node-1",
		"vega-pub-key",
		1,
		"eth-address",
		"tm-pub-key",
		"http://info.url",
		"GB",
		"Validator",
		"",
		1,
		true,
	))

	// make sure event has been processed
	time.Sleep(20 * time.Millisecond)

	server.broker.Send(events.NewVegaKeyRotationEvent(
		ctx,
		"node-1",
		"vega-pub-key",
		"new-vega-pub-key",
		10,
	))

	server.broker.Send(events.NewVegaKeyRotationEvent(
		ctx,
		"node-1",
		"new-vega-pub-key",
		"new-vega-pub-key-2",
		12,
	))

	// make sure event has been processed
	time.Sleep(20 * time.Millisecond)

	now := time.Now()
	// the broker reacts to Time events to trigger writes the data stores
	tue := events.NewTime(ctx, now)
	server.broker.Send(tue)

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	nodeID := "node-1"

	var resp *apipb.GetKeyRotationsResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.GetKeyRotations(ctx, &apipb.GetKeyRotationsRequest{})
			if err == nil && len(resp.Rotations) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Len(t, resp.Rotations, 2)
	// first rotation
	assert.Equal(t, nodeID, resp.Rotations[0].NodeId)
	assert.Equal(t, uint64(10), resp.Rotations[0].BlockHeight)
	assert.Equal(t, "vega-pub-key", resp.Rotations[0].OldPubKey)
	assert.Equal(t, "new-vega-pub-key", resp.Rotations[0].NewPubKey)
	// second rotation
	assert.Equal(t, nodeID, resp.Rotations[1].NodeId)
	assert.Equal(t, uint64(12), resp.Rotations[1].BlockHeight)
	assert.Equal(t, "new-vega-pub-key", resp.Rotations[1].OldPubKey)
	assert.Equal(t, "new-vega-pub-key-2", resp.Rotations[1].NewPubKey)
}
