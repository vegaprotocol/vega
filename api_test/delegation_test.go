// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package api_test

import (
	"context"
	"io"
	"testing"

	"code.vegaprotocol.io/vega/types/num"

	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	"code.vegaprotocol.io/vega/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waits until the delegation server has at least on subscriber
func waitForDlSubsription(ctx context.Context, ts *TestServer) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if ts.dl.GetDelegationSubscribersCount() > 0 {
				return nil
			}
		}
	}
}

func TestDelegationObserver(t *testing.T) {
	t.Run("delegations observer with an empty filter passes all", testObserveDelegationResponsesNoFilter)
	t.Run("delegations observer with a party filter passes only events matching the party", testObserveDelegationResponsesWithPartyFilter)
	t.Run("delegations observer with a node filter passes only events matching the node", testObserveDelegationResponsesWithNodeFilter)
	t.Run("delegations observer with a party/node filter passes only events matching the party and the node", testObserveDelegationResponsesWithPartyNodeFilter)
}

func testObserveDelegationResponsesNoFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveDelegationsRequest{}
	delegationEvents := []*events.DelegationBalance{
		events.NewDelegationBalance(ctx, "party1", "node1", num.NewUint(100), "1"),
		events.NewDelegationBalance(ctx, "party2", "node2", num.NewUint(400), "2"),
		events.NewDelegationBalance(ctx, "party3", "node3", num.NewUint(500), "3"),
		events.NewDelegationBalance(ctx, "party1", "node2", num.NewUint(200), "2"),
		events.NewDelegationBalance(ctx, "party2", "node4", num.NewUint(200), "4"),
	}

	testDLObserverWithFilter(t, req, delegationEvents, delegationEvents)
}

func testObserveDelegationResponsesWithPartyFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveDelegationsRequest{Party: "party1"}
	delegationEvents := []*events.DelegationBalance{
		events.NewDelegationBalance(ctx, "party1", "node1", num.NewUint(100), "1"),
		events.NewDelegationBalance(ctx, "party2", "node2", num.NewUint(400), "2"),
		events.NewDelegationBalance(ctx, "party3", "node3", num.NewUint(500), "3"),
		events.NewDelegationBalance(ctx, "party1", "node2", num.NewUint(200), "2"),
	}
	expectedEvents := []*events.DelegationBalance{
		events.NewDelegationBalance(ctx, "party1", "node1", num.NewUint(100), "1"),
		events.NewDelegationBalance(ctx, "party1", "node2", num.NewUint(200), "2"),
	}

	testDLObserverWithFilter(t, req, delegationEvents, expectedEvents)
}

func testObserveDelegationResponsesWithNodeFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveDelegationsRequest{NodeId: "node1"}
	delegationEvents := []*events.DelegationBalance{
		events.NewDelegationBalance(ctx, "party1", "node1", num.NewUint(100), "1"),
		events.NewDelegationBalance(ctx, "party3", "node2", num.NewUint(400), "2"),
		events.NewDelegationBalance(ctx, "party4", "node3", num.NewUint(500), "3"),
		events.NewDelegationBalance(ctx, "party2", "node1", num.NewUint(200), "2"),
	}
	expectedEvents := []*events.DelegationBalance{
		events.NewDelegationBalance(ctx, "party1", "node1", num.NewUint(100), "1"),
		events.NewDelegationBalance(ctx, "party2", "node1", num.NewUint(200), "2"),
	}

	testDLObserverWithFilter(t, req, delegationEvents, expectedEvents)
}

func testObserveDelegationResponsesWithPartyNodeFilter(t *testing.T) {
	ctx := context.Background()
	req := &apipb.ObserveDelegationsRequest{Party: "party1", NodeId: "node1"}
	delegationEvents := []*events.DelegationBalance{
		events.NewDelegationBalance(ctx, "party1", "node1", num.NewUint(100), "1"),
		events.NewDelegationBalance(ctx, "party3", "node2", num.NewUint(400), "2"),
		events.NewDelegationBalance(ctx, "party4", "node3", num.NewUint(500), "3"),
		events.NewDelegationBalance(ctx, "party2", "node1", num.NewUint(200), "2"),
		events.NewDelegationBalance(ctx, "party1", "node1", num.NewUint(200), "2"),
	}
	expectedEvents := []*events.DelegationBalance{
		events.NewDelegationBalance(ctx, "party1", "node1", num.NewUint(100), "1"),
		events.NewDelegationBalance(ctx, "party1", "node1", num.NewUint(200), "2"),
	}

	testDLObserverWithFilter(t, req, delegationEvents, expectedEvents)
}

func testDLObserverWithFilter(t *testing.T, req *apipb.ObserveDelegationsRequest, evts []*events.DelegationBalance, expectedEvents []*events.DelegationBalance) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	// we need to subscribe to the stream prior to publishing the events
	stream, err := client.ObserveDelegations(ctx, req)
	assert.NoError(t, err)

	// wait until the transfer response has subscribed before sending events
	err = waitForDlSubsription(ctx, server)
	require.NoError(t, err)

	for _, evt := range evts {
		server.broker.Send(evt)
	}

	var i = 0
	for i < len(expectedEvents) {
		resp, err := stream.Recv()

		// Check if the stream has finished
		if err == io.EOF {
			break
		}

		require.NotNil(t, resp)
		require.Equal(t, expectedEvents[i].Party, resp.Delegation.Party)
		require.Equal(t, expectedEvents[i].NodeID, resp.Delegation.NodeId)
		require.Equal(t, expectedEvents[i].Amount.String(), resp.Delegation.Amount)
		require.Equal(t, expectedEvents[i].EpochSeq, resp.Delegation.EpochSeq)
		i++
	}
}
