package node_test

import (
	"context"
	"fmt"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/api/node/adapters"
	nodemocks "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundRobinSelector(t *testing.T) {
	t.Run("Returns one of the healthiest node", testRoundRobinSelectorReturnsTheFirstHealthyNode)
	t.Run("Stopping the selector stops all nodes", testRoundRobinSelectorStoppingTheSelectorStopsAllNodes)
}

func testRoundRobinSelectorReturnsTheFirstHealthyNode(t *testing.T) {
	ctx := context.Background()
	log := newTestLogger(t)
	ctrl := gomock.NewController(t)

	// given
	node0 := nodemocks.NewMockNode(ctrl)
	node0.EXPECT().Host().AnyTimes().Return("node-0")

	node1 := nodemocks.NewMockNode(ctrl)
	node1.EXPECT().Host().AnyTimes().Return("node-1")

	node2 := nodemocks.NewMockNode(ctrl)
	node2.EXPECT().Host().AnyTimes().Return("node-2")

	node3 := nodemocks.NewMockNode(ctrl)
	node3.EXPECT().Host().AnyTimes().Return("node-3")

	node4 := nodemocks.NewMockNode(ctrl)
	node4.EXPECT().Host().AnyTimes().Return("node-4")

	chainID := vgrand.RandomStr(5)

	latestStats := adapters.Statistics{
		BlockHash:   vgrand.RandomStr(5),
		BlockHeight: vgrand.NewNonce(),
		ChainID:     chainID,
		VegaTime:    "123456789",
	}

	lateStats1 := adapters.Statistics{
		BlockHash:   vgrand.RandomStr(5),
		BlockHeight: vgrand.NewNonce(),
		ChainID:     chainID,
		VegaTime:    "123456780",
	}

	lateStats2 := adapters.Statistics{
		BlockHash:   vgrand.RandomStr(5),
		BlockHeight: vgrand.NewNonce(),
		ChainID:     chainID,
		VegaTime:    "123456750",
	}

	// when
	selector, err := node.NewRoundRobinSelector(log, node0, node1, node2, node3, node4)

	// then
	require.NoError(t, err)

	// given all nodes are healthy
	node0.EXPECT().Statistics(ctx).Times(10).Return(latestStats, nil)
	node1.EXPECT().Statistics(ctx).Times(10).Return(latestStats, nil)
	node2.EXPECT().Statistics(ctx).Times(10).Return(latestStats, nil)
	node3.EXPECT().Statistics(ctx).Times(10).Return(latestStats, nil)
	node4.EXPECT().Statistics(ctx).Times(10).Return(latestStats, nil)

	// This tests the round-robbin capability with healthy node only.
	for i := 0; i < 10; i++ {
		// when
		selectedNode, err := selector.Node(ctx, noReporting)

		// then it returns the next healthy node
		require.NoError(t, err)
		require.NotEmpty(t, selectedNode)
		expectedNodeHost := fmt.Sprintf("node-%d", i%5)
		assert.Equal(t, expectedNodeHost, selectedNode.Host(), fmt.Sprintf("expected %s, but got %s after %d iterations", expectedNodeHost, selectedNode.Host(), i))
	}

	node0.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node1.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node2.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node3.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node4.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)

	// when
	selectedNode, err := selector.Node(ctx, noReporting)

	// then it returns the next healthy node
	require.NoError(t, err)
	require.NotEmpty(t, selectedNode)
	assert.Equal(t, "node-0", selectedNode.Host())

	// given `node-1` and `node-2` become unhealthy,
	// and the latest node selected node is the `node-0`
	node0.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node1.EXPECT().Statistics(ctx).Times(1).Return(lateStats1, nil)
	node2.EXPECT().Statistics(ctx).Times(1).Return(lateStats2, nil)
	node3.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node4.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)

	// when
	selectedNode, err = selector.Node(ctx, noReporting)

	// then it returns the next healthy node `node-3`
	require.NoError(t, err)
	require.NotEmpty(t, selectedNode)
	assert.Equal(t, "node-3", selectedNode.Host())

	// given `node-0` and `node-4` become unhealthy,
	// and the latest node selected node is the `node-3`
	node0.EXPECT().Statistics(ctx).Times(1).Return(lateStats1, nil)
	node1.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node2.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node3.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node4.EXPECT().Statistics(ctx).Times(1).Return(lateStats1, nil)

	// when
	selectedNode, err = selector.Node(ctx, noReporting)

	// then it returns the next healthy node `node-1`
	require.NoError(t, err)
	require.NotEmpty(t, selectedNode)
	assert.Equal(t, "node-1", selectedNode.Host())

	// given `node-0`, `node-1` and `node-2` become unhealthy,
	// and the latest node selected node is the `node-4`
	node0.EXPECT().Statistics(ctx).Times(1).Return(lateStats1, nil)
	node1.EXPECT().Statistics(ctx).Times(1).Return(lateStats2, nil)
	node2.EXPECT().Statistics(ctx).Times(1).Return(lateStats2, nil)
	node3.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node4.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)

	// when
	selectedNode, err = selector.Node(ctx, noReporting)

	// then it returns the next healthy node `node-3`
	require.NoError(t, err)
	require.NotEmpty(t, selectedNode)
	assert.Equal(t, "node-3", selectedNode.Host())

	// EDGE CASE ! Ideally we would like this to not happen, but it does because
	// we can't do otherwise...
	// For more details, read the comments on the algorithm implementation.

	// given `node-0`, `node-2`, `node-4`, node-3 become unhealthy,
	// and the latest node selected node is the `node-3`
	node0.EXPECT().Statistics(ctx).Times(1).Return(lateStats2, nil)
	node1.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node2.EXPECT().Statistics(ctx).Times(1).Return(lateStats2, nil)
	node3.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node4.EXPECT().Statistics(ctx).Times(1).Return(lateStats2, nil)

	// when
	selectedNode, err = selector.Node(ctx, noReporting)

	// then it returns the next healthy node `node-4`
	require.NoError(t, err)
	require.NotEmpty(t, selectedNode)
	assert.Equal(t, "node-4", selectedNode.Host())

	// given all nodes except one don't respond except one.
	node0.EXPECT().Statistics(ctx).Times(1).Return(adapters.Statistics{}, assert.AnError)
	node1.EXPECT().Statistics(ctx).Times(1).Return(adapters.Statistics{}, assert.AnError)
	node2.EXPECT().Statistics(ctx).Times(1).Return(latestStats, nil)
	node3.EXPECT().Statistics(ctx).Times(1).Return(adapters.Statistics{}, assert.AnError)
	node4.EXPECT().Statistics(ctx).Times(1).Return(adapters.Statistics{}, assert.AnError)

	// when
	selectedNode, err = selector.Node(ctx, noReporting)

	// then it returns the next healthy node `node-2`
	require.NoError(t, err)
	require.NotEmpty(t, selectedNode)
	assert.Equal(t, "node-2", selectedNode.Host())

	// given all nodes except one don't respond except one.
	node0.EXPECT().Statistics(ctx).Times(1).Return(adapters.Statistics{}, assert.AnError)
	node1.EXPECT().Statistics(ctx).Times(1).Return(adapters.Statistics{}, assert.AnError)
	node2.EXPECT().Statistics(ctx).Times(1).Return(adapters.Statistics{}, assert.AnError)
	node3.EXPECT().Statistics(ctx).Times(1).Return(adapters.Statistics{}, assert.AnError)
	node4.EXPECT().Statistics(ctx).Times(1).Return(adapters.Statistics{}, assert.AnError)

	// when
	selectedNode, err = selector.Node(ctx, noReporting)

	// then
	require.ErrorIs(t, err, node.ErrNoHealthyNodeAvailable)
	require.Empty(t, selectedNode)
}

func testRoundRobinSelectorStoppingTheSelectorStopsAllNodes(t *testing.T) {
	// given
	log := newTestLogger(t)
	ctrl := gomock.NewController(t)

	closingHost1 := nodemocks.NewMockNode(ctrl)
	closingHost1.EXPECT().Stop().Times(1).Return(nil)

	failedClosingHost := nodemocks.NewMockNode(ctrl)
	failedClosingHost.EXPECT().Stop().Times(1).Return(assert.AnError)

	closingHost2 := nodemocks.NewMockNode(ctrl)
	closingHost2.EXPECT().Stop().Times(1).Return(nil)

	// when
	selector, err := node.NewRoundRobinSelector(log,
		closingHost1,
		failedClosingHost,
		closingHost2,
	)

	// then
	require.NoError(t, err)

	// when
	require.NotPanics(t, func() {
		selector.Stop()
	})
}
