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
			expected: "a39fa614b7b4bb0cf5819840164ca48472d1bda98a49053f891dfb004f053d2c29a60df5423927dad057f1d3d6a04c6e6d82f1bf128db5d5a7a01bcc8b70ab0e",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "d6810cef5534e232396ab0c572ca079fa41f728a20d98da3bfb59b81f183a96adee103d5f94348e9a8ce823446392109bc8bf10a27076cd0e232a1f808e0810c",
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
			expected: "a1c183d1c076c518297fa75f0fa3fddf6e5e83e76800a2efdbce36be12d4a23e2b61bce8097fea701f5a274ec89d70f92ffdd83a82a4d2c65f82b905109c3d0f",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "f11d5e0fa1c68edd1b43db30a0d02aaff8cef26c6140d30a4570615fa12a0e4e858a30b73a3763590a88614f366e9f433978653445b901e058dc07fe77595901",
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
			expected: "8c70e1bb8a74a9112ef475cdca37f63149453f5d3729164847aabf329c8932774922bb3cf41dfd7112fa8a0bdbe3f845b170e8f38406ae51fd7da00177dbc807",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "057bcb000d6961d4c8cd67f5a8a8ffac501f7077c23dcb4aab0127f1f4530e865d8627bccfb71559b9ddca0cb938b96c709fdf419d2f350ec9ab6416b888c70f",
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
