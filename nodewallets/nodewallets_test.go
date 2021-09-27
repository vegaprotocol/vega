package nodewallet_test

import (
	"testing"

	"code.vegaprotocol.io/vega/nodewallets"
	ethnw "code.vegaprotocol.io/vega/nodewallets/eth"
	vgnw "code.vegaprotocol.io/vega/nodewallets/vega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeWallet(t *testing.T) {
	t.Run("Verify node wallets succeeds", testVerifyNodeWalletsSucceeds)
	t.Run("Verify node wallets fails", testVerifyNodeWalletsFails)
}

func testVerifyNodeWalletsSucceeds(t *testing.T) {
	nw := &nodewallet.NodeWallets{
		Vega:     &vgnw.Wallet{},
		Ethereum: &ethnw.Wallet{},
	}

	assert.NoError(t, nw.Verify())
}

func testVerifyNodeWalletsFails(t *testing.T) {
	tcs := []struct {
		name        string
		expectedErr error
		nw          *nodewallet.NodeWallets
	}{
		{
			name:        "with missing Ethereum wallet",
			expectedErr: nodewallet.ErrEthereumWalletIsMissing,
			nw: &nodewallet.NodeWallets{
				Vega: &vgnw.Wallet{},
			},
		}, {
			name:        "with missing Vega wallet",
			expectedErr: nodewallet.ErrVegaWalletIsMissing,
			nw: &nodewallet.NodeWallets{
				Ethereum: &ethnw.Wallet{},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			require.EqualError(tt, tc.nw.Verify(), tc.expectedErr.Error())
		})
	}
}
