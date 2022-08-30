package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
)

type GetChainIDResult struct {
	ChainID string `json:"chainID"`
}

type GetChainID struct {
	nodeSelector NodeSelector
}

func (h *GetChainID) Handle(ctx context.Context, _ jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	currentNode, err := h.nodeSelector.Node(ctx)
	if err != nil {
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrNoHealthyNodeAvailable)
	}

	lastBlockData, err := currentNode.LastBlock(ctx)
	if err != nil {
		return nil, networkError(ErrorCodeNodeRequestFailed, ErrCouldNotGetLastBlockInformation)
	}

	return GetChainIDResult{
		ChainID: lastBlockData.ChainId,
	}, nil
}

func NewGetChainID(nodeSelector NodeSelector) *GetChainID {
	return &GetChainID{
		nodeSelector: nodeSelector,
	}
}
