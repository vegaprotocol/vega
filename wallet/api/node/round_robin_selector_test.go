package node_test

import (
	"context"
	"testing"

	apimocks "code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundRobinSelector(t *testing.T) {
	t.Run("Returns the first healthy node", testRoundRobinSelectorReturnsTheFirstHealthyNode)
	t.Run("Returns an error when no healthy node available", testRoundRobinSelectorReturnsErrorWhenNoHealthyNodeAvailable)
	t.Run("Stopping the selector stops all nodes", testRoundRobinSelectorStoppingTheSelectorStopsAllNodes)
}

func testRoundRobinSelectorReturnsTheFirstHealthyNode(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)
	ctrl := gomock.NewController(t)

	healthyHost1 := apimocks.NewMockNode(ctrl)
	healthyHost1.EXPECT().HealthCheck(ctx).Times(1).Return(nil)
	healthyHost1.EXPECT().Host().AnyTimes().Return("healthy-host-1")

	healthyHost2 := apimocks.NewMockNode(ctrl)
	healthyHost2.EXPECT().Host().AnyTimes().Return("healthy-host-2")

	unhealthyHost1 := apimocks.NewMockNode(ctrl)
	unhealthyHost1.EXPECT().HealthCheck(ctx).Times(1).Return(assert.AnError)
	unhealthyHost1.EXPECT().Host().AnyTimes().Return("unhealthy-host-1")

	unhealthyHost2 := apimocks.NewMockNode(ctrl)
	unhealthyHost2.EXPECT().HealthCheck(gomock.Any()).Times(1).Return(assert.AnError)
	unhealthyHost2.EXPECT().Host().AnyTimes().Return("unhealthy-host-2")

	unhealthyHost3 := apimocks.NewMockNode(ctrl)
	unhealthyHost3.EXPECT().Host().AnyTimes().Return("unhealthy-host-3")

	// when
	selector, err := node.NewRoundRobinSelector(log,
		unhealthyHost1,
		unhealthyHost2,
		healthyHost1,
		healthyHost2,
		unhealthyHost3,
	)

	// then
	require.NoError(t, err)

	// when
	selectedNode, err := selector.Node(ctx)

	// then
	require.NoError(t, err)
	require.NotNil(t, selectedNode)
	assert.Equal(t, healthyHost1, selectedNode)
}

func testRoundRobinSelectorReturnsErrorWhenNoHealthyNodeAvailable(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)
	ctrl := gomock.NewController(t)

	unhealthyHost1 := apimocks.NewMockNode(ctrl)
	unhealthyHost1.EXPECT().HealthCheck(ctx).Times(1).Return(assert.AnError)
	unhealthyHost1.EXPECT().Host().AnyTimes().Return("unhealthy-host-1")

	unhealthyHost2 := apimocks.NewMockNode(ctrl)
	unhealthyHost2.EXPECT().HealthCheck(gomock.Any()).Times(1).Return(assert.AnError)
	unhealthyHost2.EXPECT().Host().AnyTimes().Return("unhealthy-host-2")

	unhealthyHost3 := apimocks.NewMockNode(ctrl)
	unhealthyHost3.EXPECT().HealthCheck(gomock.Any()).Times(1).Return(assert.AnError)
	unhealthyHost3.EXPECT().Host().AnyTimes().Return("unhealthy-host-3")

	// when
	selector, err := node.NewRoundRobinSelector(log,
		unhealthyHost1,
		unhealthyHost2,
		unhealthyHost3,
	)

	// then
	require.NoError(t, err)

	// when
	selectedNode, err := selector.Node(ctx)

	// then
	require.ErrorIs(t, err, node.ErrNoHealthyNodeAvailable)
	require.Nil(t, selectedNode)
}

func testRoundRobinSelectorStoppingTheSelectorStopsAllNodes(t *testing.T) {
	// given
	log := newTestLogger(t)
	ctrl := gomock.NewController(t)

	closingHost1 := apimocks.NewMockNode(ctrl)
	closingHost1.EXPECT().Stop().Times(1).Return(nil)

	failedClosingHost := apimocks.NewMockNode(ctrl)
	failedClosingHost.EXPECT().Stop().Times(1).Return(assert.AnError)

	closingHost2 := apimocks.NewMockNode(ctrl)
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
