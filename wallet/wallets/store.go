package wallets

import (
	"fmt"

	"code.vegaprotocol.io/vega/paths"
	wstorev1 "code.vegaprotocol.io/vega/wallet/wallet/store/v1"
)

// InitialiseStore builds a wallet Store specifically for users wallets.
func InitialiseStore(vegaHome string) (*wstorev1.FileStore, error) {
	p := paths.New(vegaHome)
	return InitialiseStoreFromPaths(p)
}

// InitialiseStoreFromPaths builds a wallet Store specifically for users wallets.
func InitialiseStoreFromPaths(vegaPaths paths.Paths) (*wstorev1.FileStore, error) {
	walletsHome, err := vegaPaths.CreateDataPathFor(paths.WalletsDataHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get wallets data home path: %w", err)
	}
	return wstorev1.InitialiseStore(walletsHome)
}
