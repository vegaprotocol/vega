package abci

import "github.com/tendermint/tendermint/abci/types"

func NewResponseCheckTx(code uint32, info string) types.ResponseCheckTx {
	return types.ResponseCheckTx{
		Code: code,
		Info: info,
	}
}

func NewResponseDeliverTx(code uint32, info string) types.ResponseDeliverTx {
	return types.ResponseDeliverTx{
		Code: code,
		Info: info,
	}
}
