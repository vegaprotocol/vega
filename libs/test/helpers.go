package test

import (
	"context"
	"path/filepath"
	"sync"

	vgcontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
)

func RandomPath() string {
	return filepath.Join("/tmp", "vega_tests", vgrand.RandomStr(10))
}

func OnlyOnce(f func()) func() {
	var once sync.Once

	return func() {
		once.Do(f)
	}
}

func VegaContext(chainId string, blockHeight int64) context.Context {
	return vgcontext.WithChainID(
		vgcontext.WithTraceID(
			vgcontext.WithBlockHeight(context.Background(), blockHeight),
			vgcrypto.RandomHash(),
		),
		chainId)
}
