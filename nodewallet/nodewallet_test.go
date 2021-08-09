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
	"code.vegaprotocol.io/vega/nodewallet/eth"
	"code.vegaprotocol.io/vega/nodewallet/eth/mocks"
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
	t.Run("test devInit success", testDevInitSuccess)
	t.Run("verify success", testVerifySuccess)
	t.Run("verify failure", testVerifyFailure)
	t.Run("new failure invalid store path", testNewFailureInvalidStorePath)
	t.Run("new failure missing required wallets", testNewFailureMissingRequiredWallets)
	t.Run("new failure invalidPassphrase", testNewFailureInvalidPassphrase)
	t.Run("import new wallet", testImportNewWallet)
}

func testInitSuccess(t *testing.T) {
	rootDir := rootDir()

	err := nodewallet.Initialise(rootDir, "somepassphrase")
	assert.NoError(t, err)

	assert.NoError(t, os.RemoveAll(rootDir))
}

func testDevInitSuccess(t *testing.T) {
	rootDir := rootDir()
	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	err := nodewallet.DevInit(rootDir, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()

	ethClient.EXPECT().ChainID(gomock.Any()).Times(1).Return(big.NewInt(42), nil)
	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, rootDir)
	require.NoError(t, err)
	assert.NotNil(t, nw)

	w, ok := nw.Get(nodewallet.Ethereum)
	assert.NotNil(t, w)
	assert.True(t, ok)
	assert.Equal(t, string(nodewallet.Ethereum), w.Chain())

	w1, ok := nw.Get(nodewallet.Vega)
	assert.NotNil(t, w1)
	assert.True(t, ok)
	assert.Equal(t, string(nodewallet.Vega), w1.Chain())

	assert.NoError(t, os.RemoveAll(rootDir))
}

func testVerifySuccess(t *testing.T) {
	rootDir := rootDir()
	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	err := nodewallet.DevInit(rootDir, "somepassphrase")
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	ethClient.EXPECT().ChainID(gomock.Any()).Times(1).Return(big.NewInt(42), nil)
	defer ctrl.Finish()

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, rootDir)
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
	rootDir := rootDir()
	cfg := nodewallet.Config{
		Level: encoding.LogLevel{},
	}

	err := nodewallet.Initialise(rootDir, "somepassphrase")
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	ethClient := mocks.NewMockETHClient(ctrl)
	defer ctrl.Finish()
	ethClient.EXPECT().ChainID(gomock.Any()).Times(1).Return(big.NewInt(42), nil)

	nw, err := nodewallet.New(logging.NewTestLogger(), cfg, "somepassphrase", ethClient, rootDir)
	require.NoError(t, err)
	assert.NotNil(t, nw)

	// now generate an eth wallet
	fileName, err := eth.DevInit(rootDir, "ethpassphrase")
	require.NoError(t, err)
	assert.NotEmpty(t, fileName)

	// import this new wallet
	filePath := filepath.Join(rootDir, fileName)
	err = nw.Import(string(nodewallet.Ethereum), "somepassphrase", "ethpassphrase", filePath)
	require.NoError(t, err)

	assert.NoError(t, os.RemoveAll(rootDir))
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
