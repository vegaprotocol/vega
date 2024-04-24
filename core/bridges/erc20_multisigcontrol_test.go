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
			expected: "a2c61b473f15a1729e8593d65748e7a9813102e0d7304598af556525206db599fb79b9750349c6cb564a2f3ecdf233dd19b1598302e0cb91218adff1c609ac09",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "aa79559d350a9b139d04d7883b7ec26b3948bba503fddcc55f8a868a69ef48dad32ffb4233a041401e482e71232fc339aa6deffda31bcd978596a6a0a6d64b0c",
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
			expected: "7bdc018935610f23667b31d4eee248160ab39caa1e70ad20da49bf8971d5a16b30f71a09d9aaf5b532defdb7710d85c226e98cb90a49bc4b4401b33f3c5a1601",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "f86654970ab8aa7b8f1ac72cd1349cd667acd21b7ff2078653d488f3ab65a446df1b4878692d7f07e2f0111bed069fd7cf5c32f07ae88ed059624480cd0edd07",
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
			expected: "98ea2303c68dbb0a88bdb7dad8c6e2db9698cd992667399a378e682dbdf16e74a9d304a32e36b48de81c0e99449a7a37c1a7ef94af1e85aa88a808f8d7126c0c",
		},
		{
			name:     "v2 scheme",
			v1:       false,
			expected: "e17efd360ce488a7299175473f257544391e3823db314e31cc69e6ae2730ead994e89bfab5813ea1379c4b6e499d131308ebe516ba6142f9f77479083685020b",
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
