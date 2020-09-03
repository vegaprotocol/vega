package abci

import abci "github.com/tendermint/tendermint/abci/types"

func NewResponseCheckTx(code uint32) abci.ResponseCheckTx {
	return abci.ResponseCheckTx{
		Code: code,
	}
}

func NewResponseDeliverTx(code uint32) abci.ResponseDeliverTx {
	return abci.ResponseDeliverTx{
		Code: code,
	}
}
