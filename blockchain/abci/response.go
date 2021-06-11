package abci

import "github.com/tendermint/tendermint/abci/types"

func NewResponseCheckTx(code uint32, info string) types.ResponseCheckTx {
	return types.ResponseCheckTx{
		Code: code,
		Info: info,
	}
}

func NewResponseCheckTxError(code uint32, err error) types.ResponseCheckTx {
	return types.ResponseCheckTx{
		Code: code,
		Info: err.Error(),
		Data: []byte(err.Error()),
	}
}

func NewResponseDeliverTx(code uint32, info string) types.ResponseDeliverTx {
	return types.ResponseDeliverTx{
		Code: code,
		Info: info,
	}
}

func NewResponseDeliverTxError(code uint32, err error) types.ResponseDeliverTx {
	return types.ResponseDeliverTx{
		Code: code,
		Info: err.Error(),
		Data: []byte(err.Error()),
	}
}
