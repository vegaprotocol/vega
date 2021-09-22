//go:build !race
// +build !race

package nodewallet_test

import (
	"os"
	"path/filepath"
	"testing"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/nodewallet/eth/mocks"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newVegaPaths() (paths.Paths, func()) {
	path := filepath.Join("/tmp", "vegatests", "nodewallet", vgrand.RandomStr(10))
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return paths.NewPaths(path), func() { _ = os.RemoveAll(path) }
}

func TestNodeWallet(t *testing.T) {
	t.Run("test init success as new node wallet", testInitSuccess)
	t.Run("test generation success", testGenerationSuccess)
	t.Run("verify success", testVerifySuccess)
	t.Run("verify failure", testVerifyFailure)
	t.Run("new failure missing required wallets", testNewFailureMissingRequiredWallets)
	t.Run("new failure invalidPassphrase", testNewFailureInvalidPassphrase)
	t.Run("import new wallet", testImportNewWallet)
	t.Run("show success", testShowSuccess)
}

func testInitSuccess(t *testing.T) {
	vegaPaths, cleanupFn := newVegaPaths()
	defer cleanupFn()

	defer cleanupFn()

	_, err := nodewallet.InitialiseLoader(vegaPaths, "somepassphrase")
	assert.NoError(t, err)
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
			vegaPaths, cleanupFn := newVegaPaths()
			defer cleanupFn()

			cfg := nodewallet.Config{
				Level: encoding.LogLevel{},
			}

			_, err := nodewallet.InitialiseLoader(vegaPaths, "somepassphrase")
			require.NoError(tt, err)

			ctrl := gomock.NewController(tt)
			ethClient := mocks.NewMockETHClient(ctrl)
			defer ctrl.Finish()

			nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, vegaPaths)
			require.NoError(tt, err)
			assert.NotNil(tt, nw)

			_, err = nw.Generate(c.chain, "somepassphrase", "eth-passphrase")
			require.NoError(tt, err)

			w, ok := nw.Get(nodewallet.Blockchain(c.chain))
			assert.NotNil(tt, w)
			assert.True(tt, ok)
			assert.Equal(tt, c.chain, w.Chain())

			defer cleanupFn()
		})
	}
}

func testVerifySuccess(t *testing.T) {
	vegaPaths, cleanupFn := newVegaPaths()
	defer cleanupFn()

	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	_, err := nodewallet.InitialiseLoader(vegaPaths, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, vegaPaths)
	require.NoError(t, err)
	assert.NotNil(t, nw)

	data, err := nw.Generate(string(nodewallet.Ethereum), "somepassphrase", "eth-passphrase")
	require.NoError(t, err)
	assert.NotEmpty(t, data["walletFilePath"])

	data, err = nw.Generate(string(nodewallet.Vega), "somepassphrase", "vega-somepassphrase")
	require.NoError(t, err)
	assert.NotEmpty(t, data["mnemonic"])
	assert.NotEmpty(t, data["walletFilePath"])

	err = nw.Verify()
	assert.NoError(t, err)
}

func testVerifyFailure(t *testing.T) {
	nw := &nodewallet.Service{}

	err := nw.Verify()
	assert.Error(t, err)
}

func testNewFailureMissingRequiredWallets(t *testing.T) {
	vegaPaths, cleanupFn := newVegaPaths()
	defer cleanupFn()

	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	_, err := nodewallet.InitialiseLoader(vegaPaths, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, vegaPaths)
	require.NoError(t, err)

	assert.EqualError(t, nw.Verify(), "required wallet for vega chain is missing")
}

func testImportNewWallet(t *testing.T) {
	vegaPaths, cleanupFn := newVegaPaths()
	defer cleanupFn()

	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	_, err := nodewallet.InitialiseLoader(vegaPaths, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, vegaPaths)
	require.NoError(t, err)
	assert.NotNil(t, nw)

	// now generate an eth wallet
	ethWalletPath := filepath.Join("/tmp", "vegatests", "nodewallet", vgrand.RandomStr(10))
	defer os.RemoveAll(ethWalletPath)
	ks := keystore.NewKeyStore(ethWalletPath, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount("ethpassphrase")
	require.NoError(t, err)

	// import this new wallet
	_, err = nw.Import(string(nodewallet.Ethereum), "somepassphrase", "ethpassphrase", acc.URL.Path)
	require.NoError(t, err)
}

func testNewFailureInvalidPassphrase(t *testing.T) {
	vegaPaths, cleanupFn := newVegaPaths()
	defer cleanupFn()

	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	_, err := nodewallet.InitialiseLoader(vegaPaths, "somepassphrase")
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "notthesamepassphrase", ethClient, vegaPaths)
	assert.EqualError(t, err, "couldn't load node wallet store: wrong passphrase")
	assert.Nil(t, nw)
}

func testShowSuccess(t *testing.T) {
	vegaPaths, cleanupFn := newVegaPaths()
	defer cleanupFn()

	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	_, err := nodewallet.InitialiseLoader(vegaPaths, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, vegaPaths)
	require.NoError(t, err)
	assert.NotNil(t, nw)

	data, err := nw.Generate(string(nodewallet.Ethereum), "somepassphrase", "eth-passphrase")
	require.NoError(t, err)
	assert.NotEmpty(t, data["walletFilePath"])

	data, err = nw.Generate(string(nodewallet.Vega), "somepassphrase", "vega-passphrase")
	require.NoError(t, err)
	assert.NotEmpty(t, data["mnemonic"])
	assert.NotEmpty(t, data["walletFilePath"])

	configs := nw.Show()

	assert.Equal(t, "vega", configs["vega"].Chain)
	assert.NotEmpty(t, configs["vega"].Name)
	assert.Equal(t, "vega-passphrase", configs["vega"].Passphrase)
	assert.Equal(t, "ethereum", configs["ethereum"].Chain)
	assert.NotEmpty(t, configs["ethereum"].Name)
	assert.Equal(t, "eth-passphrase", configs["ethereum"].Passphrase)
}
