// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package verify_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"code.vegaprotocol.io/vega/cmd/vega/commands/verify"
	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/genesis"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/validators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	t.Run("verify default genesis", testVerifyDefaultGenesis)
	t.Run("verify ERC20 assets", testVerifyERC20Assets)
	t.Run("verify builtin assets", testVerifyBuiltinAssets)
	t.Run("verify netparams", testVerifyNetworkParams)
	t.Run("verify validators", TestVerifyValidators)
	t.Run("verify unknown appstate field", testUnknownAppstateField)
}

func testVerifyDefaultGenesis(t *testing.T) {
	testFile := getFileFromAppstate(t, genesis.DefaultState())

	cmd := verify.GenesisCmd{}
	assert.NoError(t, cmd.Execute([]string{testFile}))
}

func testVerifyBuiltinAssets(t *testing.T) {
	cmd := verify.GenesisCmd{}

	// LP stake not a bignum
	gs := genesis.DefaultState()
	gs.Assets["FAILURE"] = assets.AssetDetails{
		Quantum: "FAILURE",
		Source: &assets.Source{
			BuiltinAsset: &assets.BuiltinAsset{
				MaxFaucetAmountMint: "100",
			},
		},
	}

	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Max faucet amount not a bignum
	gs = genesis.DefaultState()
	gs.Assets["FAILURE"] = assets.AssetDetails{
		Quantum: "100",
		Source: &assets.Source{
			BuiltinAsset: &assets.BuiltinAsset{
				MaxFaucetAmountMint: "FAILURE",
			},
		},
	}

	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Completely Valid
	gs = genesis.DefaultState()
	gs.Assets["FAILURE"] = assets.AssetDetails{
		Quantum: "100",
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
	gs := genesis.DefaultState()
	gs.Assets["tooshort"] = assets.AssetDetails{
		Quantum: "100",
		Source: &assets.Source{
			Erc20: &assets.Erc20{
				ContractAddress: "0xBC944ba38753A6fCAdd634Be98379330dbaB3Eb8",
			},
		},
	}
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Invalid contract address
	gs = genesis.DefaultState()
	gs.Assets["b4f2726571fbe8e33b442dc92ed2d7f0d810e21835b7371a7915a365f07ccd9b"] = assets.AssetDetails{
		Quantum: "100",
		Source: &assets.Source{
			Erc20: &assets.Erc20{
				ContractAddress: "invalid",
			},
		},
	}
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Completely valid
	gs = genesis.DefaultState()
	gs.Assets["b4f2726571fbe8e33b442dc92ed2d7f0d810e21835b7371a7915a365f07ccd9b"] = assets.AssetDetails{
		Quantum: "100",
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
	gs := genesis.DefaultState()
	gs.NetParams["NOTREAL"] = "something"
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Check for network parameter with bad value
	gs = genesis.DefaultState()
	gs.NetParams["snapshot.interval.length"] = "always"
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Check for invalid checkpoint overwrite network parameter
	gs = genesis.DefaultState()
	gs.NetParamsOverwrite = []string{"NOTREAL"}
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Check for deprecated parameter in genesis
	gs = genesis.DefaultState()
	for k := range netparams.Deprecated {
		gs.NetParams[k] = "hello"
	}
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))
}

func TestVerifyValidators(t *testing.T) {
	cmd := verify.GenesisCmd{}

	valid := validators.ValidatorData{
		ID:              "eb2374c1e8e746cb5fbda66ee69eba0c2c551bea8793afe8c5a239b9763d14bf",
		VegaPubKey:      "Adf2e74b372be36f6373ea9c2c4cf496310852228c54867726dbb77528b35761",
		VegaPubKeyIndex: 4,
		EthereumAddress: "0xF0a9b5d3a00b53362F9b73892124743BAaE526c4",
		TmPubKey:        "2D2TXGN2GD4GTCQV9sbrXw7RVb3td7S4pWq6v3wIpvI=",
	}
	// Valid validator information
	gs := genesis.DefaultState()
	gs.Validators[valid.TmPubKey] = valid
	assert.NoError(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Mismatch TM key
	gs = genesis.DefaultState()
	gs.Validators["WRONG"] = valid
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// Invalid pubkey index
	gs = genesis.DefaultState()
	invalid := valid
	invalid.VegaPubKeyIndex = 0
	gs.Validators[valid.TmPubKey] = invalid
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// invalid ID
	gs = genesis.DefaultState()
	invalid = valid
	invalid.ID = "too short"
	gs.Validators[valid.TmPubKey] = invalid
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))

	// invalid pubkey
	gs = genesis.DefaultState()
	invalid = valid
	invalid.VegaPubKey = "too short"
	gs.Validators[valid.TmPubKey] = invalid
	assert.Error(t, cmd.Execute([]string{getFileFromAppstate(t, gs)}))
}

func testUnknownAppstateField(t *testing.T) {
	cmd := verify.GenesisCmd{}
	gs := genesis.DefaultState()

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

func getFileFromAppstate(t *testing.T, gs genesis.State) string {
	t.Helper()

	testFile := filepath.Join(t.TempDir(), "genesistest.json")

	genesis := struct {
		AppState        genesis.State `json:"app_state"`
		ConsensusParams struct {
			Block struct {
				TimeIotaMs string `json:"time_iota_ms"`
			} `json:"block"`
		} `json:"consensus_params"`
	}{AppState: gs}
	genesis.ConsensusParams.Block.TimeIotaMs = "1"
	// marshall it
	file, _ := json.MarshalIndent(genesis, "", " ")

	err := os.WriteFile(testFile, file, 0o644)

	// write to file
	require.NoError(t, err)
	return testFile
}
