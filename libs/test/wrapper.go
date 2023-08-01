package test

import (
	"context"

	vgcontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
)

func VegaContext(chainId string, blockHeight int64) context.Context {
	return vgcontext.WithChainID(
		vgcontext.WithTraceID(
			vgcontext.WithBlockHeight(context.Background(), blockHeight),
			vgcrypto.RandomHash(),
		),
		chainId)
}
