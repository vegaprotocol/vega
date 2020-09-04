package abci

import "github.com/tendermint/tendermint/abci/types"

func NewResponseCheckTx(code uint32) types.ResponseCheckTx {
	return types.ResponseCheckTx{
		Code: code,
	}
}

func NewResponseDeliverTx(code uint32) types.ResponseDeliverTx {
	return types.ResponseDeliverTx{
		Code: code,
	}
}
