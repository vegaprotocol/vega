package verify_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/cmd/vega/verify"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/validators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	t.Run("verify default genesis", testVerifyDefaultGenesis)
	t.Run("verify ERC20 assets", testVerifyERC20Assets)
	t.Run("verify builtin assets", testVerifyBuiltinAssets)
	t.Run("verify netparams", testVerifyNetworkParams)
	t.Run("verify netparams", testVerifyValidators)
	t.Run("verify unknown appstate field", testUnknownAppstateField)
}

func testVerifyDefaultGenesis(t *testing.T) {
	testFile := getFileFromAppstate(t, genesis.DefaultGenesisState())

	cmd := verify.GenesisCmd{}
	assert.NoError(t, cmd.Execute([]string{testFile}))
}

func testVerifyBuiltinAssets(t *testing.T) {
	cmd := verify.GenesisCmd{}

	// LP stake not a bignum
	gs := genesis.DefaultGenesisState()
	gs.Assets["FAILURE"] = assets.AssetDetails{
		TotalSupply: "100",
		Quantum:     "FAILURE",
		Source: &assets.Source{
			BuiltinAsset: &assets.BuiltinAsset{
				MaxFaucetAmountMint: "100",
			},
		},
	}

	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Max faucet amount not a bignum
	gs = genesis.DefaultGenesisState()
	gs.Assets["FAILURE"] = assets.AssetDetails{
		TotalSupply: "100",
		Quantum:     "100",
		Source: &assets.Source{
			BuiltinAsset: &assets.BuiltinAsset{
				MaxFaucetAmountMint: "FAILURE",
			},
		},
	}

	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Completely Valid
	gs = genesis.DefaultGenesisState()
	gs.Assets["FAILURE"] = assets.AssetDetails{
		TotalSupply: "100",
		Quantum:     "100",
		Source: &assets.Source{
			BuiltinAsset: &assets.BuiltinAsset{
				MaxFaucetAmountMint: "100",
			},
		},
	}

	assert.NoError(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))
}

func testVerifyERC20Assets(t *testing.T) {
	cmd := verify.GenesisCmd{}

	// Invalid ID
	gs := genesis.DefaultGenesisState()
	gs.Assets["tooshort"] = assets.AssetDetails{
		TotalSupply: "100",
		Quantum:     "100",
		Source: &assets.Source{
			Erc20: &assets.Erc20{
				ContractAddress: "0xBC944ba38753A6fCAdd634Be98379330dbaB3Eb8",
			},
		},
	}
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Invalid contract address
	gs = genesis.DefaultGenesisState()
	gs.Assets["b4f2726571fbe8e33b442dc92ed2d7f0d810e21835b7371a7915a365f07ccd9b"] = assets.AssetDetails{
		TotalSupply: "100",
		Quantum:     "100",
		Source: &assets.Source{
			Erc20: &assets.Erc20{
				ContractAddress: "invalid",
			},
		},
	}
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Completely valid
	gs = genesis.DefaultGenesisState()
	gs.Assets["b4f2726571fbe8e33b442dc92ed2d7f0d810e21835b7371a7915a365f07ccd9b"] = assets.AssetDetails{
		TotalSupply: "100",
		Quantum:     "100",
		Source: &assets.Source{
			Erc20: &assets.Erc20{
				ContractAddress: "0xF0a9b5d3a00b53362F9b73892124743BAaE526c4",
			},
		},
	}

	assert.NoError(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))
}

func testVerifyNetworkParams(t *testing.T) {
	cmd := verify.GenesisCmd{}

	// Check for invalid network parameter
	gs := genesis.DefaultGenesisState()
	gs.NetParams["NOTREAL"] = "something"
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Check for network parameter with bad value
	gs = genesis.DefaultGenesisState()
	gs.NetParams["snapshot.interval.length"] = "always"
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Check for invalid checkpoint overwrite network parameter
	gs = genesis.DefaultGenesisState()
	gs.NetParamsOverwrite = []string{"NOTREAL"}
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))
}

func testVerifyValidators(t *testing.T) {
	cmd := verify.GenesisCmd{}

	valid := validators.ValidatorData{
		ID:              "eb2374c1e8e746cb5fbda66ee69eba0c2c551bea8793afe8c5a239b9763d14bf",
		VegaPubKey:      "adf2e74b372be36f6373ea9c2c4cf496310852228c54867726dbb77528b35761",
		VegaPubKeyIndex: 4,
		EthereumAddress: "0xF0a9b5d3a00b53362F9b73892124743BAaE526c4",
		TmPubKey:        "2D2TXGN2GD4GTCQV9sbrXw7RVb3td7S4pWq6v3wIpvI=",
	}
	// Valid validator information
	gs := genesis.DefaultGenesisState()
	gs.Validators[valid.TmPubKey] = valid
	assert.NoError(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Mismatch TM key
	gs = genesis.DefaultGenesisState()
	gs.Validators["WRONG"] = valid
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Invalid pubkey index
	gs = genesis.DefaultGenesisState()
	invalid := valid
	invalid.VegaPubKeyIndex = 0
	gs.Validators[valid.TmPubKey] = invalid
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// invalid ID
	gs = genesis.DefaultGenesisState()
	invalid = valid
	invalid.ID = "too short"
	gs.Validators[valid.TmPubKey] = invalid
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// invalid pubkey
	gs = genesis.DefaultGenesisState()
	invalid = valid
	invalid.VegaPubKey = "too short"
	gs.Validators[valid.TmPubKey] = invalid
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))
}

func testUnknownAppstateField(t *testing.T) {
	cmd := verify.GenesisCmd{}
	gs := genesis.DefaultGenesisState()

	// Marshall and unmarshal unstructured so we can add an unknown field
	var unstructured map[string]interface{}
	b, err := json.Marshal(gs)
	require.NoError(t, err)

	err = json.Unmarshal(b, &unstructured)
	require.NoError(t, err)

	unstructured["unknownfield"] = "unknownvalue"
	n, err := json.Marshal(unstructured)
	require.NoError(t, err)

	testFile := filepath.Join(t.TempDir(), "genesistest.json")

	genesis := struct {
		AppState json.RawMessage `json:"app_state"`
	}{AppState: json.RawMessage(n)}

	// marshall it
	file, _ := json.MarshalIndent(genesis, "", " ")
	err = os.WriteFile(testFile, file, 0o644)
	require.NoError(t, err)

	// expected failure given unknown field
	assert.Error(t, cmd.Execute([]string{testFile}))
}

func getFileFromAppstate(t *testing.T, gs genesis.GenesisState) string {
	t.Helper()

	testFile := filepath.Join(t.TempDir(), "genesistest.json")

	genesis := struct {
		AppState genesis.GenesisState `json:"app_state"`
	}{AppState: gs}
	// marshall it
	file, _ := json.MarshalIndent(genesis, "", " ")
	err := os.WriteFile(testFile, file, 0o644)

	// write to file
	require.NoError(t, err)
	return testFile
}
