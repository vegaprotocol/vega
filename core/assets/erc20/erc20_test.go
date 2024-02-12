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

package erc20_test

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets/erc20"
	ethnw "code.vegaprotocol.io/vega/core/nodewallets/eth"
	"code.vegaprotocol.io/vega/core/nodewallets/registry"
	"code.vegaprotocol.io/vega/core/types"
	vcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ed25519"
)

const (
	privKey       = "9feb9cbee69c1eeb30db084544ff8bf92166bf3fddefa6a021b458b4de04c66758a127387b1dff15b71fd7d0a9fd104ed75da4aac549efd5d149051ea57cefaf"
	pubKey        = "58a127387b1dff15b71fd7d0a9fd104ed75da4aac549efd5d149051ea57cefaf"
	tokenID       = "af077ace8cbf3179f826f2d3485b812f6efef07d913f2ed02f295360dd78b30e"
	ethPartyAddr  = "0x1ebe188952ab6035adad21ea1c4f64fd2eac60e1"
	bridgeAddress = "0xcB84d72e61e383767C4DFEb2d8ff7f4FB89abc6e"
)

var token = &types.AssetDetails{
	Name:     "VEGA",
	Symbol:   "VEGA",
	Decimals: 18,
	Quantum:  num.DecimalFromFloat(1),
	Source: &types.AssetDetailsErc20{
		ERC20: &types.ERC20{
			ContractAddress:   "0x1FaA74E181092A97Fecc923015293ce57eE1208A",
			WithdrawThreshold: num.NewUint(1000),
			LifetimeLimit:     num.NewUint(42),
		},
	},
}

type testERC20 struct {
	*erc20.ERC20
	wallet    *ethnw.Wallet
	ethClient testEthClient
}

func newTestERC20(t *testing.T) *testERC20 {
	t.Helper()
	wallet := ethnw.NewWallet(testWallet{})
	ethClient := testEthClient{}
	erc20Token, err := erc20.New(tokenID, token, wallet, ethClient)
	assert.NoError(t, err)

	return &testERC20{
		ERC20:     erc20Token,
		wallet:    wallet,
		ethClient: ethClient,
	}
}

func TestERC20Signatures(t *testing.T) {
	t.Run("withdraw_asset", testWithdrawAsset)
	t.Run("list_asset", testListAsset)
}

func testWithdrawAsset(t *testing.T) {
	token := newTestERC20(t)
	now := time.Unix(10000, 0)
	msg, sig, err := token.SignWithdrawal(
		num.NewUint(42),
		ethPartyAddr,
		big.NewInt(84),
		now,
	)

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.NotNil(t, sig)
	assert.True(t, verifySignature(msg, sig))
	assert.Equal(t,
		"68154aa30a66d8546a338e2f50ac3e0bde710975755562e12c8508c5e4e43aa741b98d1f7384d8cf6a33e86fc1ed6f833ad627a9fb9b5a56aaaf0024511a2402",
		hex.EncodeToString(sig),
	)
}

func testListAsset(t *testing.T) {
	token := newTestERC20(t)
	msg, sig, err := token.SignListAsset()

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.NotNil(t, sig)
	assert.True(t, verifySignature(msg, sig))
	assert.Equal(t,
		"e6048f597145d7d1e1ddfe41abf9ae950e9b6e93598c8b1e4fe2d9af8493b240a4d85322eb40c6bf76b0eac2481fa42014956f10a38675769b0c995e191d650b",
		hex.EncodeToString(sig),
	)
}

type testEthClient struct {
	bind.ContractBackend
}

func (testEthClient) ConfiguredChainID() string {
	return "1"
}

func (testEthClient) HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error) {
	return nil, nil
}

func (testEthClient) CollateralBridgeAddress() ethcommon.Address {
	return ethcommon.HexToAddress(bridgeAddress)
}

func (testEthClient) CurrentHeight(context.Context) (uint64, error) { return 100, nil }
func (testEthClient) ConfirmationsRequired() uint64                 { return 1 }

type testWallet struct{}

func (testWallet) Cleanup() error { return nil }
func (testWallet) Name() string   { return "eth" }
func (testWallet) Chain() string  { return "eth" }
func (testWallet) Sign(data []byte) ([]byte, error) {
	priv, _ := hex.DecodeString(privKey)
	return ed25519.Sign(ed25519.PrivateKey(priv), data), nil
}
func (testWallet) Algo() string                                  { return "eth" }
func (testWallet) Version() (string, error)                      { return "1", nil }
func (testWallet) PubKey() vcrypto.PublicKey                     { return vcrypto.PublicKey{} }
func (testWallet) Reload(d registry.EthereumWalletDetails) error { return nil }

func verifySignature(msg, sig []byte) bool {
	pub, _ := hex.DecodeString(pubKey)
	hash := ethcrypto.Keccak256(msg)
	return ed25519.Verify(ed25519.PublicKey(pub), hash, sig)
}
