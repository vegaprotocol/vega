package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
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

func TestNewNodeEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	server.broker.Send(events.NewEpochEvent(ctx, &types.Epoch{Seq: 12}))
	server.broker.Send(events.NewValidatorRanking(ctx, "12", "node-1", "1", "1", "1", "VALIDATOR_STATUS_PENDING", "VALIDATOR_STATUS_PENDING", 10))
	time.Sleep(20 * time.Millisecond) // we want to make sure the ranking gets sent first and the we keep it at hand until the node event comes through
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

	now := time.Now()
	// the broker reacts to Time events to trigger writes the data stores
	tue := events.NewTime(ctx, now)
	server.broker.Send(tue)

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	nodeID := "node-1"
	var resp *apipb.GetNodeByIDResponse
	var err error
loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.GetNodeByID(ctx, &apipb.GetNodeByIDRequest{Id: nodeID})
			if err == nil && resp.Node != nil {
				break loop
			}
		}
	}

	assert.NotNil(t, resp.Node)
	assert.NotNil(t, resp.Node.RankingScore)

	allNodes, err := client.GetNodes(ctx, &apipb.GetNodesRequest{})
	assert.NoError(t, err)
	assert.Len(t, allNodes.Nodes, 1)

	// move the epoch along to one where the node hasn't got a ranking i.e it has been removed and check it is not returned in the node list
	server.broker.Send(events.NewEpochEvent(ctx, &types.Epoch{Seq: 13}))
	var resp2 *apipb.GetEpochResponse

loop2:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp2, err = client.GetEpoch(ctx, &apipb.GetEpochRequest{Id: 13})
			if err == nil && resp2.Epoch != nil {
				break loop2
			}
		}
	}
	assert.NotNil(t, resp2.Epoch)
	assert.Len(t, resp2.Epoch.Validators, 0)
	_, err = client.GetNodeByID(ctx, &apipb.GetNodeByIDRequest{Id: nodeID})
	assert.Error(t, err) // because its not found

	allNodes, err = client.GetNodes(ctx, &apipb.GetNodesRequest{})
	assert.NoError(t, err)
	assert.Len(t, allNodes.Nodes, 0)
}
