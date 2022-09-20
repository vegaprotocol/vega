package api_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	nodemocks "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChainID(t *testing.T) {
	t.Run("Getting chain ID succeeds", testGettingChainIDSucceeds)
	t.Run("No healthy node available does not return the chain ID", testNoHealthyNodeAvailableDoesNotReturnChainID)
	t.Run("Failing to get the last block does not return the chain ID", testFailingToGetLastBlockDoesNotReturnChainID)
}

func testGettingChainIDSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	expectedChainID := vgrand.RandomStr(5)

	// setup
	handler := newGetChainIDHandler(t)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(&apipb.LastBlockHeightResponse{
		ChainId: expectedChainID,
	}, nil)

	// when
	result, errorDetails := handler.handle(t, ctx)

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, expectedChainID, result.ChainID)
}

func testNoHealthyNodeAvailableDoesNotReturnChainID(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newGetChainIDHandler(t)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeRequestFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrNoHealthyNodeAvailable.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingToGetLastBlockDoesNotReturnChainID(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newGetChainIDHandler(t)
	handler.nodeSelector.EXPECT().Node(ctx).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeRequestFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrCouldNotGetLastBlockInformation.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

type GetChainIDHandler struct {
	*api.GetChainID
	nodeSelector *nodemocks.MockSelector
	node         *nodemocks.MockNode
}

func (h *GetChainIDHandler) handle(t *testing.T, ctx context.Context) (api.GetChainIDResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, nil)
	if rawResult != nil {
		result, ok := rawResult.(api.GetChainIDResult)
		if !ok {
			t.Fatal("GetChainID handler result is not a GetChainIDResult")
		}
		return result, err
	}
	return api.GetChainIDResult{}, err
}

func newGetChainIDHandler(t *testing.T) *GetChainIDHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	nodeSelector := nodemocks.NewMockSelector(ctrl)
	node := nodemocks.NewMockNode(ctrl)

	return &GetChainIDHandler{
		GetChainID:   api.NewGetChainID(nodeSelector),
		nodeSelector: nodeSelector,
		node:         node,
	}
}
