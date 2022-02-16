package bridges_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/bridges"
	"code.vegaprotocol.io/vega/types/num"

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
	t.Run("set minimum deposit", testMinimumDeposit)
	t.Run("set maximum deposit", testMaximumDeposit)
}

func testListAsset(t *testing.T) {
	signer := testSigner{}
	bridge := bridges.NewERC20Logic(signer, erc20BridgeAddr)
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
	assert.Equal(t,
		"7df8b88552c2f981e64b13f1ce3ee5dcb71e8f59ec057010b7b469120afff7d479f234714785cfc605230dfb2d17f9cc7858143196a13f357ce008e3f3f78a00",
		sig.Signature.Hex(),
	)
}

func testRemoveAsset(t *testing.T) {
	signer := testSigner{}
	bridge := bridges.NewERC20Logic(signer, erc20BridgeAddr)
	sig, err := bridge.RemoveAsset(
		erc20AssetAddr,
		num.NewUint(42),
	)

	assert.NoError(t, err)
	assert.NotNil(t, sig.Message)
	assert.NotNil(t, sig.Signature)
	assert.True(t, signer.Verify(sig.Message, sig.Signature))
	assert.Equal(t,
		"9012eb20763500caf1a4d7640470449c7220872d7136e17c70231c269051cf80e08760d60850578ebf494e24610a54225c7d994f15f57d9f451e8f717eb3f904",
		sig.Signature.Hex(),
	)
}

func testWithdrawAsset(t *testing.T) {
	signer := testSigner{}
	bridge := bridges.NewERC20Logic(signer, erc20BridgeAddr)
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
	assert.Equal(t,
		"0ff08571ab504acdce063a5a5a00dd8878d64ccb09ea6887aacd1fd41b517cd13f4e12edfaa4d06fef5d24087ba9e7c980532daa0a6f1fa329b8d75961f4ab03",
		sig.Signature.Hex(),
	)
}

func testMinimumDeposit(t *testing.T) {
	signer := testSigner{}
	bridge := bridges.NewERC20Logic(signer, erc20BridgeAddr)
	sig, err := bridge.SetDepositMinimum(
		erc20AssetAddr,
		num.NewUint(42),  // amount
		num.NewUint(100), // nonce
	)

	assert.NoError(t, err)
	assert.NotNil(t, sig.Message)
	assert.NotNil(t, sig.Signature)
	assert.True(t, signer.Verify(sig.Message, sig.Signature))
	assert.Equal(t,
		"2c8526e9c662dda56ea4f911d890338b8f9048ae6c147321f17b98678d37fa4186f41e120afcc3b312d884a88e0e39fbf4800e1d8a151be7ce85a49e6cbbdd01",
		sig.Signature.Hex(),
	)
}

func testMaximumDeposit(t *testing.T) {
	signer := testSigner{}
	bridge := bridges.NewERC20Logic(signer, erc20BridgeAddr)
	sig, err := bridge.SetDepositMaximum(
		erc20AssetAddr,
		num.NewUint(42),
		num.NewUint(100), // nonce
	)

	assert.NoError(t, err)
	assert.NotNil(t, sig.Message)
	assert.NotNil(t, sig.Signature)
	assert.True(t, signer.Verify(sig.Message, sig.Signature))
	assert.Equal(t,
		"7b5131822184055173ac46eb1b78b4c01a5d239ec555e151820b43de4a92c93c7575bc8e43573faf200b08f3d88354c7743f20917b1da91d3d0f44f829ff1c0f",
		sig.Signature.Hex(),
	)
}
