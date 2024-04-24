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

package bridges_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/bridges"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

const (
	erc20BridgeAddr  = "0xcB84d72e61e383767C4DFEb2d8ff7f4FB89abc6e"
	erc20AssetVegaID = "e74758a6708a866cd9262aae09170087f1b8afd7187fca752cd640cb93915fad"
	erc20AssetAddr   = "0x1FaA74E181092A97Fecc923015293ce57eE1208A"
	ethPartyAddr     = "0x1ebe188952ab6035adad21ea1c4f64fd2eac60e1"
)

func TestERC20Logic(t *testing.T) {
	t.Run("list asset", testListAsset)
	t.Run("remove asset", testRemoveAsset)
	t.Run("withdraw asset", testWithdrawAsset)
}

func testListAsset(t *testing.T) {
	tcs := []struct {
		name     string
		v1       bool
		expected string
	}{
		{
			name:     "v1 scheme",
			v1:       true,
			expected: "7df8b88552c2f981e64b13f1ce3ee5dcb71e8f59ec057010b7b469120afff7d479f234714785cfc605230dfb2d17f9cc7858143196a13f357ce008e3f3f78a00",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "03d8d648da4402bebd096f067cebf3e3b70f2c4e1cad6ca9eb757f554b6ca9efb84010887aeef543cf72cb5d78a741d0683befc6f5e0ca2d0347832232af610c",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			signer := testSigner{}
			bridge := bridges.NewERC20Logic(signer, erc20BridgeAddr, chainID, tc.v1)
			sig, err := bridge.ListAsset(
				erc20AssetAddr,
				erc20AssetVegaID,
				num.NewUint(10),
				num.NewUint(42),
				num.NewUint(42),
			)

			assert.NoError(t, err)
			assert.NotNil(t, sig.Message)
			assert.NotNil(t, sig.Signature)
			assert.True(t, signer.Verify(sig.Message, sig.Signature))
			assert.Equal(t, tc.expected, sig.Signature.Hex())
		})
	}
}

func testRemoveAsset(t *testing.T) {
	tcs := []struct {
		name     string
		v1       bool
		expected string
	}{
		{
			name:     "v1 scheme",
			v1:       true,
			expected: "9012eb20763500caf1a4d7640470449c7220872d7136e17c70231c269051cf80e08760d60850578ebf494e24610a54225c7d994f15f57d9f451e8f717eb3f904",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "aa07e175a9a4c3dcb0f5dcbd24cc6636e699ee6a1daa9a80267cec8f0be130b86465fa56296743879f56d94d6be64a0b10b76bcee40d0d09ec078b2814b89500",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			signer := testSigner{}
			bridge := bridges.NewERC20Logic(signer, erc20BridgeAddr, chainID, tc.v1)
			sig, err := bridge.RemoveAsset(
				erc20AssetAddr,
				num.NewUint(42),
			)

			assert.NoError(t, err)
			assert.NotNil(t, sig.Message)
			assert.NotNil(t, sig.Signature)
			assert.True(t, signer.Verify(sig.Message, sig.Signature))
			assert.Equal(t, tc.expected, sig.Signature.Hex())
		})
	}
}

func testWithdrawAsset(t *testing.T) {
	tcs := []struct {
		name     string
		v1       bool
		expected string
	}{
		{
			name:     "v1 scheme",
			v1:       true,
			expected: "0ff08571ab504acdce063a5a5a00dd8878d64ccb09ea6887aacd1fd41b517cd13f4e12edfaa4d06fef5d24087ba9e7c980532daa0a6f1fa329b8d75961f4ab03",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "9f2d7ec17059fd5d4697337a46899f73681dece748ea1342b3be24b5f34f0b934ad448f7e9bd3a113102d46d8433dd26458cf06c3fd7a1622d086faab1a77b08",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			signer := testSigner{}
			bridge := bridges.NewERC20Logic(signer, erc20BridgeAddr, chainID, tc.v1)
			sig, err := bridge.WithdrawAsset(
				erc20AssetAddr,
				num.NewUint(42), // amount
				ethPartyAddr,
				time.Unix(1000, 0),
				num.NewUint(1000), // nonce
			)

			assert.NoError(t, err)
			assert.NotNil(t, sig.Message)
			assert.NotNil(t, sig.Signature)
			assert.True(t, signer.Verify(sig.Message, sig.Signature))
			assert.Equal(t, tc.expected, sig.Signature.Hex())
		})
	}
}
