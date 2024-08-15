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
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/core/bridges"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

const (
	privKey = "9feb9cbee69c1eeb30db084544ff8bf92166bf3fddefa6a021b458b4de04c66758a127387b1dff15b71fd7d0a9fd104ed75da4aac549efd5d149051ea57cefaf"
	pubKey  = "58a127387b1dff15b71fd7d0a9fd104ed75da4aac549efd5d149051ea57cefaf"

	chainID = "31337"
)

func TestERC20MultiSigControl(t *testing.T) {
	t.Run("set threshold", testSetThreshold)
	t.Run("add signer", testAddSigner)
	t.Run("remove signer", testRemoveSigner)
}

func testSetThreshold(t *testing.T) {
	tcs := []struct {
		name     string
		v1       bool
		expected string
	}{
		{
			name:     "v1 scheme",
			v1:       true,
			expected: "537eee3a9151d3f9a0a076c9521d3ca014efaefb20fd9be1b36c1e2475897f82b79b0de759b24a52a1a66c272c85628acd182a3cef2ae92b91a595f8d8123c06",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "e1bc702b74ca31f08d1d3534b6b147f641cf94713a809dd5b7ed5cb6e61b1892f1d2327ef14194bac488f41a661b805c6ca2f3daba7108eee46949229a126c0e",
		},
	}

	for _, tc := range tcs {
		signer := testSigner{}
		bridge := bridges.NewERC20MultiSigControl(signer, chainID, tc.v1)
		sig, err := bridge.SetThreshold(
			1000,
			"0x1FaA74E181092A97Fecc923015293ce57eE1208A",
			num.NewUint(42),
		)

		assert.NoError(t, err)
		assert.NotNil(t, sig.Message)
		assert.NotNil(t, sig.Signature)
		assert.True(t, signer.Verify(sig.Message, sig.Signature))
		assert.Equal(t, tc.expected, sig.Signature.Hex())
	}
}

func testAddSigner(t *testing.T) {
	tcs := []struct {
		name     string
		v1       bool
		expected string
	}{
		{
			name:     "v1 scheme",
			v1:       true,
			expected: "52201353f14009c7638a6460af9340871de8d34a198d4957a142aa8098922829d4963eff0e6951fdecfe04566c865225b4c41dd2393b40d4de76fb9819142d0c",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "67e5a71935eaf134f4cbfadbefa96e510c97d6198accfcec76b046ee46ecd59fe4ae6f371973a8f4be7a67dd20b38c421b0f24061f1c95fe1e9322d04997e407",
		},
	}

	for _, tc := range tcs {
		signer := testSigner{}
		bridge := bridges.NewERC20MultiSigControl(signer, chainID, tc.v1)
		sig, err := bridge.AddSigner(
			"0xE20c747a7389B7De2c595658277132f188A074EE",
			"0x1FaA74E181092A97Fecc923015293ce57eE1208A",
			num.NewUint(42),
		)

		assert.NoError(t, err)
		assert.NotNil(t, sig.Message)
		assert.NotNil(t, sig)
		assert.True(t, signer.Verify(sig.Message, sig.Signature))

		assert.Equal(t, tc.expected, sig.Signature.Hex())
	}
}

func testRemoveSigner(t *testing.T) {
	tcs := []struct {
		name     string
		v1       bool
		expected string
	}{
		{
			name:     "v1 scheme",
			v1:       true,
			expected: "4bf8057aa87a4ec5049766b0eb40426c9ba0464cda2ce203d16c590f3153657689143161908923b0c6ab32dec57c4c5aca7e4aaef24b8d22362413908868ea00",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "648f08b810bcbe589b79f7476f48de6e2c3528fbb059427f3876745dec51128952210d5e509e307d0dc2f6ba67f65bda7a28d96a8d31a365d04398cdde78150b",
		},
	}

	for _, tc := range tcs {
		signer := testSigner{}
		bridge := bridges.NewERC20MultiSigControl(signer, chainID, tc.v1)
		sig, err := bridge.RemoveSigner(
			"0xE20c747a7389B7De2c595658277132f188A074EE",
			"0x1FaA74E181092A97Fecc923015293ce57eE1208A",
			num.NewUint(42),
		)

		assert.NoError(t, err)
		assert.NotNil(t, sig.Message)
		assert.NotNil(t, sig)
		assert.True(t, signer.Verify(sig.Message, sig.Signature))
		assert.Equal(t, tc.expected, sig.Signature.Hex())
	}
}

type testSigner struct{}

func (s testSigner) Algo() string { return "ed25519" }

func (s testSigner) Sign(msg []byte) ([]byte, error) {
	priv, _ := hex.DecodeString(privKey)

	return ed25519.Sign(ed25519.PrivateKey(priv), msg), nil
}

func (s testSigner) Verify(msg, sig []byte) bool {
	pub, _ := hex.DecodeString(pubKey)
	hash := crypto.Keccak256(msg)

	return ed25519.Verify(ed25519.PublicKey(pub), hash, sig)
}
