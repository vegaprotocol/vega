package bridges_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/bridges"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

const (
	privKey = "9feb9cbee69c1eeb30db084544ff8bf92166bf3fddefa6a021b458b4de04c66758a127387b1dff15b71fd7d0a9fd104ed75da4aac549efd5d149051ea57cefaf"
	pubKey  = "58a127387b1dff15b71fd7d0a9fd104ed75da4aac549efd5d149051ea57cefaf"
)

func TestERC20MultiSigControl(t *testing.T) {
	t.Run("set threshold", testSetThreshold)
	t.Run("add signer", testAddSigner)
	t.Run("remove signer", testRemoveSigner)
}

func testSetThreshold(t *testing.T) {
	signer := testSigner{}
	bridge := bridges.NewERC20MultiSigControl(signer)
	sig, err := bridge.SetThreshold(
		1000,
		"0x1FaA74E181092A97Fecc923015293ce57eE1208A",
		num.NewUint(42),
	)

	assert.NoError(t, err)
	assert.NotNil(t, sig.Message)
	assert.NotNil(t, sig.Signature)
	assert.True(t, signer.Verify(sig.Message, sig.Signature))
	assert.Equal(t,
		"a2c61b473f15a1729e8593d65748e7a9813102e0d7304598af556525206db599fb79b9750349c6cb564a2f3ecdf233dd19b1598302e0cb91218adff1c609ac09",
		sig.Signature.Hex(),
	)

}

func testAddSigner(t *testing.T) {
	signer := testSigner{}
	bridge := bridges.NewERC20MultiSigControl(signer)
	sig, err := bridge.AddSigner(
		"0xE20c747a7389B7De2c595658277132f188A074EE",
		"0x1FaA74E181092A97Fecc923015293ce57eE1208A",
		num.NewUint(42),
	)

	assert.NoError(t, err)
	assert.NotNil(t, sig.Message)
	assert.NotNil(t, sig)
	assert.True(t, signer.Verify(sig.Message, sig.Signature))

	assert.Equal(t,
		"7bdc018935610f23667b31d4eee248160ab39caa1e70ad20da49bf8971d5a16b30f71a09d9aaf5b532defdb7710d85c226e98cb90a49bc4b4401b33f3c5a1601",
		sig.Signature.Hex(),
	)
}

func testRemoveSigner(t *testing.T) {
	signer := testSigner{}
	bridge := bridges.NewERC20MultiSigControl(signer)
	sig, err := bridge.RemoveSigner(
		"0xE20c747a7389B7De2c595658277132f188A074EE",
		"0x1FaA74E181092A97Fecc923015293ce57eE1208A",
		num.NewUint(42),
	)

	assert.NoError(t, err)
	assert.NotNil(t, sig.Message)
	assert.NotNil(t, sig)
	assert.True(t, signer.Verify(sig.Message, sig.Signature))
	assert.Equal(t,
		"98ea2303c68dbb0a88bdb7dad8c6e2db9698cd992667399a378e682dbdf16e74a9d304a32e36b48de81c0e99449a7a37c1a7ef94af1e85aa88a808f8d7126c0c",
		sig.Signature.Hex(),
	)
}

type testSigner struct{}

func (s testSigner) Sign(msg []byte) ([]byte, error) {
	priv, _ := hex.DecodeString(privKey)

	return ed25519.Sign(ed25519.PrivateKey(priv), msg), nil
}

func (s testSigner) Verify(msg, sig []byte) bool {
	pub, _ := hex.DecodeString(pubKey)
	hash := crypto.Keccak256(msg)

	return ed25519.Verify(ed25519.PublicKey(pub), hash, sig)
}
