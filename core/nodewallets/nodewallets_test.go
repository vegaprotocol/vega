// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package nodewallets_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/nodewallets"
	ethnw "code.vegaprotocol.io/vega/core/nodewallets/eth"
	vgnw "code.vegaprotocol.io/vega/core/nodewallets/vega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeWallet(t *testing.T) {
	t.Run("Verify node wallets succeeds", testVerifyNodeWalletsSucceeds)
	t.Run("Verify node wallets fails", testVerifyNodeWalletsFails)
}

func testVerifyNodeWalletsSucceeds(t *testing.T) {
	nw := &nodewallets.NodeWallets{
		Vega:       &vgnw.Wallet{},
		Ethereum:   &ethnw.Wallet{},
		Tendermint: &nodewallets.TendermintPubkey{},
	}

	assert.NoError(t, nw.Verify())
}

func testVerifyNodeWalletsFails(t *testing.T) {
	tcs := []struct {
		name        string
		expectedErr error
		nw          *nodewallets.NodeWallets
	}{
		{
			name:        "with missing Ethereum wallet",
			expectedErr: nodewallets.ErrEthereumWalletIsMissing,
			nw: &nodewallets.NodeWallets{
				Vega: &vgnw.Wallet{},
			},
		}, {
			name:        "with missing Vega wallet",
			expectedErr: nodewallets.ErrVegaWalletIsMissing,
			nw: &nodewallets.NodeWallets{
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
