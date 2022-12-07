package api_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	nodemocks "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
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
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{
		ChainID: expectedChainID,
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
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrNoHealthyNodeAvailable.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

func testFailingToGetLastBlockDoesNotReturnChainID(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newGetChainIDHandler(t)
	handler.nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(handler.node, nil)
	handler.node.EXPECT().LastBlock(ctx).Times(1).Return(types.LastBlock{}, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx)

	// then
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, "Network error", errorDetails.Message)
	assert.Equal(t, api.ErrCouldNotGetLastBlockInformation.Error(), errorDetails.Data)
	assert.Empty(t, result)
}

type GetChainIDHandler struct {
	*api.ClientGetChainID
	nodeSelector *nodemocks.MockSelector
	node         *nodemocks.MockNode
}

func (h *GetChainIDHandler) handle(t *testing.T, ctx context.Context) (api.ClientGetChainIDResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx)
	if rawResult != nil {
		result, ok := rawResult.(api.ClientGetChainIDResult)
		if !ok {
			t.Fatal("ClientGetChainID handler result is not a ClientGetChainIDResult")
		}
		return result, err
	}
	return api.ClientGetChainIDResult{}, err
}

func newGetChainIDHandler(t *testing.T) *GetChainIDHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	nodeSelector := nodemocks.NewMockSelector(ctrl)
	node := nodemocks.NewMockNode(ctrl)

	return &GetChainIDHandler{
		ClientGetChainID: api.NewGetChainID(nodeSelector),
		nodeSelector:     nodeSelector,
		node:             node,
	}
}
