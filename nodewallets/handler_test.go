// +build !race

package nodewallets_test

import (
	"testing"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/shared/paths"
	vgtesting "code.vegaprotocol.io/vega/libs/testing"
	"code.vegaprotocol.io/vega/nodewallets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("Getting node wallets succeeds", testHandlerGettingNodeWalletsSucceeds)
	t.Run("Getting node wallets with wrong registry passphrase fails", testHandlerGettingNodeWalletsWithWrongRegistryPassphraseFails)
	t.Run("Getting Ethereum wallet succeeds", testHandlerGettingEthereumWalletSucceeds)
	t.Run("Getting Ethereum wallet succeeds", testHandlerGettingEthereumWalletWithWrongRegistryPassphraseFails)
	t.Run("Getting Vega wallet succeeds", testHandlerGettingVegaWalletSucceeds)
	t.Run("Getting Vega wallet succeeds", testHandlerGettingVegaWalletWithWrongRegistryPassphraseFails)
	t.Run("Generating Ethereum wallet succeeds", testHandlerGeneratingEthereumWalletSucceeds)
	t.Run("Generating an already existing Ethereum wallet fails", testHandlerGeneratingAlreadyExistingEthereumWalletFails)
	t.Run("Generating Ethereum wallet with overwrite succeeds", testHandlerGeneratingEthereumWalletWithOverwriteSucceeds)
	t.Run("Generating Vega wallet succeeds", testHandlerGeneratingVegaWalletSucceeds)
	t.Run("Generating an already existing Vega wallet fails", testHandlerGeneratingAlreadyExistingVegaWalletFails)
	t.Run("Generating Vega wallet with overwrite succeeds", testHandlerGeneratingVegaWalletWithOverwriteSucceeds)
	t.Run("Importing Ethereum wallet succeeds", testHandlerImportingEthereumWalletSucceeds)
	t.Run("Importing an already existing Ethereum wallet fails", testHandlerImportingAlreadyExistingEthereumWalletFails)
	t.Run("Importing Ethereum wallet with overwrite succeeds", testHandlerImportingEthereumWalletWithOverwriteSucceeds)
	t.Run("Importing Vega wallet succeeds", testHandlerImportingVegaWalletSucceeds)
	t.Run("Importing an already existing Vega wallet fails", testHandlerImportingAlreadyExistingVegaWalletFails)
	t.Run("Importing Vega wallet with overwrite succeeds", testHandlerImportingVegaWalletWithOverwriteSucceeds)
}

func testHandlerGettingNodeWalletsSucceeds(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletsPass := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// setup
	createTestNodeWallets(vegaPaths, registryPass, walletsPass)

	// when
	nw, err := nodewallets.GetNodeWallets(config, vegaPaths, registryPass)

	// assert
	require.NoError(t, err)
	require.NotNil(t, nw)
	require.NotNil(t, nw.Ethereum)
	require.NotNil(t, nw.Vega)
}

func testHandlerGettingNodeWalletsWithWrongRegistryPassphraseFails(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	wrongRegistryPass := vgrand.RandomStr(10)
	walletsPass := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// setup
	createTestNodeWallets(vegaPaths, registryPass, walletsPass)

	// when
	nw, err := nodewallets.GetNodeWallets(config, vegaPaths, wrongRegistryPass)

	// assert
	require.Error(t, err)
	assert.Nil(t, nw)
}

func testHandlerGettingEthereumWalletSucceeds(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletsPass := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// setup
	createTestNodeWallets(vegaPaths, registryPass, walletsPass)

	// when
	wallet, err := nodewallets.GetEthereumWallet(config.ETH, vegaPaths, registryPass)

	// assert
	require.NoError(t, err)
	assert.NotNil(t, wallet)
}

func testHandlerGettingEthereumWalletWithWrongRegistryPassphraseFails(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	wrongRegistryPass := vgrand.RandomStr(10)
	walletsPass := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// setup
	createTestNodeWallets(vegaPaths, registryPass, walletsPass)

	// when
	wallet, err := nodewallets.GetEthereumWallet(config.ETH, vegaPaths, wrongRegistryPass)

	// assert
	require.Error(t, err)
	assert.Nil(t, wallet)
}

func testHandlerGettingVegaWalletSucceeds(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletsPass := vgrand.RandomStr(10)

	// setup
	createTestNodeWallets(vegaPaths, registryPass, walletsPass)

	// when
	wallet, err := nodewallets.GetVegaWallet(vegaPaths, registryPass)

	// then
	require.NoError(t, err)
	assert.NotNil(t, wallet)
}

func testHandlerGettingVegaWalletWithWrongRegistryPassphraseFails(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	wrongRegistryPass := vgrand.RandomStr(10)
	walletsPass := vgrand.RandomStr(10)

	// setup
	createTestNodeWallets(vegaPaths, registryPass, walletsPass)

	// when
	wallet, err := nodewallets.GetVegaWallet(vegaPaths, wrongRegistryPass)

	// assert
	require.Error(t, err)
	assert.Nil(t, wallet)
}

func testHandlerGeneratingEthereumWalletSucceeds(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// when
	data, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass, false)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, data["registryFilePath"])
	assert.NotEmpty(t, data["walletFilePath"])
}

func testHandlerGeneratingAlreadyExistingEthereumWalletFails(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass1 := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// when
	data1, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass1, false)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, data1["registryFilePath"])
	assert.NotEmpty(t, data1["walletFilePath"])

	// given
	walletPass2 := vgrand.RandomStr(10)

	// when
	data2, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass2, false)

	// then
	require.EqualError(t, err, nodewallets.ErrEthereumWalletAlreadyExists.Error())
	assert.Empty(t, data2)
}

func testHandlerGeneratingEthereumWalletWithOverwriteSucceeds(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass1 := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// when
	data1, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass1, false)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, data1["registryFilePath"])
	assert.NotEmpty(t, data1["walletFilePath"])

	// given
	walletPass2 := vgrand.RandomStr(10)

	// when
	data2, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass2, true)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, data2["registryFilePath"])
	assert.Equal(t, data1["registryFilePath"], data2["registryFilePath"])
	assert.NotEmpty(t, data2["walletFilePath"])
	assert.NotEqual(t, data1["walletFilePath"], data2["walletFilePath"])
}

func testHandlerGeneratingVegaWalletSucceeds(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)

	// when
	data, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass, false)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, data["registryFilePath"])
	assert.NotEmpty(t, data["walletFilePath"])
	assert.NotEmpty(t, data["mnemonic"])
}

func testHandlerGeneratingAlreadyExistingVegaWalletFails(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass1 := vgrand.RandomStr(10)

	// when
	data1, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass1, false)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, data1["registryFilePath"])
	assert.NotEmpty(t, data1["walletFilePath"])
	assert.NotEmpty(t, data1["mnemonic"])

	// given
	walletPass2 := vgrand.RandomStr(10)

	// when
	data2, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass2, false)

	// then
	require.EqualError(t, err, nodewallets.ErrVegaWalletAlreadyExists.Error())
	assert.Empty(t, data2)
}

func testHandlerGeneratingVegaWalletWithOverwriteSucceeds(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass1 := vgrand.RandomStr(10)

	// when
	data1, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass1, false)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, data1["registryFilePath"])
	assert.NotEmpty(t, data1["walletFilePath"])

	// given
	walletPass2 := vgrand.RandomStr(10)

	// when
	data2, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass2, true)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, data2["registryFilePath"])
	assert.Equal(t, data1["registryFilePath"], data2["registryFilePath"])
	assert.NotEmpty(t, data2["walletFilePath"])
	assert.NotEqual(t, data1["walletFilePath"], data2["walletFilePath"])
	assert.NotEmpty(t, data2["mnemonic"])
	assert.NotEqual(t, data1["mnemonic"], data2["mnemonic"])
}

func testHandlerImportingEthereumWalletSucceeds(t *testing.T) {
	// given
	genVegaPaths, genCleanupFn := vgtesting.NewVegaPaths()
	defer genCleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// when
	genData, err := nodewallets.GenerateEthereumWallet(config.ETH, genVegaPaths, registryPass, walletPass, false)

	// then
	require.NoError(t, err)

	// given
	importVegaPaths, importCleanupFn := vgtesting.NewVegaPaths()
	defer importCleanupFn()

	// when
	importData, err := nodewallets.ImportEthereumWallet(config.ETH, importVegaPaths, registryPass, walletPass, "", genData["walletFilePath"], false)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, importData["registryFilePath"])
	assert.NotEqual(t, genData["registryFilePath"], importData["registryFilePath"])
	assert.NotEmpty(t, importData["walletFilePath"])
	assert.NotEqual(t, genData["walletFilePath"], importData["walletFilePath"])
}

func testHandlerImportingAlreadyExistingEthereumWalletFails(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// when
	genData, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass, false)

	// then
	require.NoError(t, err)

	// when
	importData, err := nodewallets.ImportEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass, "", genData["walletFilePath"], false)

	// then
	require.EqualError(t, err, nodewallets.ErrEthereumWalletAlreadyExists.Error())
	assert.Empty(t, importData)
}

func testHandlerImportingEthereumWalletWithOverwriteSucceeds(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()

	// when
	genData, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass, false)

	// then
	require.NoError(t, err)

	// when
	importData, err := nodewallets.ImportEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass, "", genData["walletFilePath"], true)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, genData["registryFilePath"])
	assert.Equal(t, importData["registryFilePath"], genData["registryFilePath"])
	assert.NotEmpty(t, genData["walletFilePath"])
	assert.Equal(t, importData["walletFilePath"], genData["walletFilePath"])
}

func testHandlerImportingVegaWalletSucceeds(t *testing.T) {
	// given
	genVegaPaths, genCleanupFn := vgtesting.NewVegaPaths()
	defer genCleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)

	// when
	genData, err := nodewallets.GenerateVegaWallet(genVegaPaths, registryPass, walletPass, false)

	// then
	require.NoError(t, err)

	// given
	importVegaPaths, importCleanupFn := vgtesting.NewVegaPaths()
	defer importCleanupFn()

	// when
	importData, err := nodewallets.ImportVegaWallet(importVegaPaths, registryPass, walletPass, genData["walletFilePath"], false)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, importData["registryFilePath"])
	assert.NotEqual(t, genData["registryFilePath"], importData["registryFilePath"])
	assert.NotEmpty(t, importData["walletFilePath"])
	assert.NotEqual(t, genData["walletFilePath"], importData["walletFilePath"])
}

func testHandlerImportingAlreadyExistingVegaWalletFails(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)

	// when
	genData, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass, false)

	// then
	require.NoError(t, err)

	// when
	importData, err := nodewallets.ImportVegaWallet(vegaPaths, registryPass, walletPass, genData["walletFilePath"], false)

	// then
	require.EqualError(t, err, nodewallets.ErrVegaWalletAlreadyExists.Error())
	assert.Empty(t, importData)
}

func testHandlerImportingVegaWalletWithOverwriteSucceeds(t *testing.T) {
	// given
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletPass := vgrand.RandomStr(10)

	// when
	genData, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass, false)

	// then
	require.NoError(t, err)

	// when
	importData, err := nodewallets.ImportVegaWallet(vegaPaths, registryPass, walletPass, genData["walletFilePath"], true)

	// then
	require.NoError(t, err)
	assert.NotEmpty(t, importData["registryFilePath"])
	assert.Equal(t, genData["registryFilePath"], importData["registryFilePath"])
	assert.NotEmpty(t, importData["walletFilePath"])
	assert.NotEqual(t, genData["walletFilePath"], importData["walletFilePath"])
}

func createTestNodeWallets(vegaPaths paths.Paths, registryPass, walletPass string) {
	config := nodewallets.NewDefaultConfig()

	if _, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, registryPass, walletPass, false); err != nil {
		panic("couldn't generate Ethereum node wallet for tests")
	}

	if _, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass, false); err != nil {
		panic("couldn't generate Vega node wallet for tests")
	}
}
