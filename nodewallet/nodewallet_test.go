// +build !race

package nodewallet_test

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/nodewallet/eth/mocks"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rootDir() string {
	path := filepath.Join("/tmp", "vegatests", "nodewallet", crypto.RandomStr(10))
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return path
}

func TestNodeWallet(t *testing.T) {
	t.Run("test init success as new node wallet", testInitSuccess)
	t.Run("test generation success", testGenerationSuccess)
	t.Run("verify success", testVerifySuccess)
	t.Run("verify failure", testVerifyFailure)
	t.Run("new failure invalid store path", testNewFailureInvalidStorePath)
	t.Run("new failure missing required wallets", testNewFailureMissingRequiredWallets)
	t.Run("new failure invalidPassphrase", testNewFailureInvalidPassphrase)
	t.Run("import new wallet", testImportNewWallet)
	t.Run("show success", testShowSuccess)
}

func testInitSuccess(t *testing.T) {
	rootDir := rootDir()

	err := nodewallet.Initialise(rootDir, "somepassphrase")
	assert.NoError(t, err)

	assert.NoError(t, os.RemoveAll(rootDir))
}

func testGenerationSuccess(t *testing.T) {
	cases := []struct {
		name  string
		chain string
	}{
		{
			name:  "for Ethereum wallet",
			chain: "ethereum",
		}, {
			name:  "for Vega wallet",
			chain: "vega",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(tt *testing.T) {
			rootDir := rootDir()
			cfg := nodewallet.Config{
				Level: encoding.LogLevel{},
			}

			err := nodewallet.Initialise(rootDir, "somepassphrase")
			require.NoError(tt, err)

			ctrl := gomock.NewController(tt)
			ethClient := mocks.NewMockETHClient(ctrl)
			ethClient.EXPECT().ChainID(gomock.Any()).Times(1).Return(big.NewInt(42), nil)
			defer ctrl.Finish()

			nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, rootDir)
			require.NoError(tt, err)
			assert.NotNil(tt, nw)

			err = nw.Generate(string(nodewallet.Ethereum), "somepassphrase", "eth-passphrase")
			require.NoError(tt, err)

			w, ok := nw.Get(nodewallet.Ethereum)
			assert.NotNil(tt, w)
			assert.True(tt, ok)
			assert.Equal(tt, string(nodewallet.Ethereum), w.Chain())

			assert.NoError(tt, os.RemoveAll(rootDir))
		})
	}
}

func testVerifySuccess(t *testing.T) {
	rootDir := rootDir()
	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	err := nodewallet.Initialise(rootDir, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	ethClient.EXPECT().ChainID(gomock.Any()).Times(1).Return(big.NewInt(42), nil)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, rootDir)
	require.NoError(t, err)
	assert.NotNil(t, nw)

	err = nw.Generate(string(nodewallet.Ethereum), "somepassphrase", "eth-passphrase")
	require.NoError(t, err)

	err = nw.Generate(string(nodewallet.Vega), "somepassphrase", "vega-somepassphrase")
	require.NoError(t, err)

	err = nw.Verify()
	assert.NoError(t, err)
	assert.NoError(t, os.RemoveAll(rootDir))
}

func testVerifyFailure(t *testing.T) {
	nw := &nodewallet.Service{}

	err := nw.Verify()
	assert.Error(t, err)
}

func testNewFailureInvalidStorePath(t *testing.T) {
	rootDir := rootDir()
	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, rootDir)
	assert.Error(t, err)
	assert.Nil(t, nw)
}

func testNewFailureMissingRequiredWallets(t *testing.T) {
	rootDir := rootDir()
	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	err := nodewallet.Initialise(rootDir, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	ethClient.EXPECT().ChainID(gomock.Any()).Times(1).Return(big.NewInt(42), nil)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, rootDir)
	require.NoError(t, err)

	assert.EqualError(t, nw.Verify(), "missing required wallet for vega chain")
	assert.NoError(t, os.RemoveAll(rootDir))
}

func testImportNewWallet(t *testing.T) {
	walletRootDir := rootDir()
	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	err := nodewallet.Initialise(walletRootDir, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()
	ethClient.EXPECT().ChainID(gomock.Any()).Times(1).Return(big.NewInt(42), nil)

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, walletRootDir)
	require.NoError(t, err)
	assert.NotNil(t, nw)

	// now generate an eth wallet
	ethWalletPath := rootDir()
	ks := keystore.NewKeyStore(ethWalletPath, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount("ethpassphrase")
	require.NoError(t, err)

	// import this new wallet
	err = nw.Import(string(nodewallet.Ethereum), "somepassphrase", "ethpassphrase", acc.URL.Path)
	require.NoError(t, err)

	assert.NoError(t, os.RemoveAll(walletRootDir))
	assert.NoError(t, os.RemoveAll(ethWalletPath))
}

func testNewFailureInvalidPassphrase(t *testing.T) {
	rootDir := rootDir()
	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	err := nodewallet.Initialise(rootDir, "somepassphrase")
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "notthesamepassphrase", ethClient, rootDir)
	assert.EqualError(t, err, "unable to load store: unable to decrypt store file (cipher: message authentication failed)")
	assert.Nil(t, nw)
	assert.NoError(t, os.RemoveAll(rootDir))
}

func testShowSuccess(t *testing.T) {
	rootDir := rootDir()
	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	err := nodewallet.Initialise(rootDir, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	ethClient.EXPECT().ChainID(gomock.Any()).Times(1).Return(big.NewInt(42), nil)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, rootDir)
	require.NoError(t, err)
	assert.NotNil(t, nw)

	err = nw.Generate(string(nodewallet.Ethereum), "somepassphrase", "eth-passphrase")
	require.NoError(t, err)

	err = nw.Generate(string(nodewallet.Vega), "somepassphrase", "vega-passphrase")
	require.NoError(t, err)

	configs := nw.Show()

	assert.Equal(t, "vega", configs["vega"].Chain)
	assert.NotEmpty(t, configs["vega"].Name)
	assert.Equal(t, "vega-passphrase", configs["vega"].Passphrase)
	assert.Equal(t, "ethereum", configs["ethereum"].Chain)
	assert.NotEmpty(t, configs["ethereum"].Name)
	assert.Equal(t, "eth-passphrase", configs["ethereum"].Passphrase)
}
