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

package verify

import (
	"bytes"
	"encoding/json"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/netparams"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

// These types are copies of the ones in the engines that read the genesis file appstate
// but the double-book keeping allows us to know when a change to the genesis state occurs.
type validator struct {
	ID               string `json:"id"`
	VegaPubKey       string `json:"vega_pub_key"`
	VegaPubKeyIndex  uint32 `json:"vega_pub_key_index"`
	EthereumAddress  string `json:"ethereum_address"`
	TMPubKey         string `json:"tm_pub_key"`
	InfoURL          string `json:"info_url"`
	Country          string `json:"country"`
	Name             string `json:"name"`
	AvatarURL        string `json:"avatar_url"`
	FromEpoch        uint64 `json:"from_epoch"`
	SubmitterAddress string `json:"submitter_address"`
}

type asset struct {
	Name     string
	Symbol   string
	Decimals uint64
	Quantum  string `json:"quantum"`
	Source   *struct {
		BuiltInAsset *struct {
			MaxFaucetAmountMint string `json:"max_faucet_amount_mint"`
		} `json:"builtin_asset,omitempty"`
		ERC20 *struct {
			ContractAddress   string `json:"contract_address"`
			LifetimeLimit     string `json:"lifetime_limit"`
			WithdrawThreshold string `json:"withdraw_threshold"`
			ChainID           string `json:"chain_id"`
		} `json:"erc20,omitempty"`
	}
}

type appState struct {
	Network *struct {
		ReplayAttackThreshold *int `json:"replay_attack_threshold"`
	} `json:"network"`
	NetworkParameters            map[string]string    `json:"network_parameters"`
	Validators                   map[string]validator `json:"validators"`
	Assets                       map[string]asset     `json:"assets"`
	NetworkParametersCPOverwrite []string             `json:"network_parameters_checkpoint_overwrite"`
	NetworkLimits                *json.RawMessage     `json:"network_limits"`
	Checkpoint                   *json.RawMessage     `json:"checkpoint"`
}

type GenesisCmd struct{}

func (opts *GenesisCmd) Execute(params []string) error {
	return verifier(params, verifyGenesis)
}

type noopBroker struct{}

func (n noopBroker) Send(e events.Event) {}

func (noopBroker) SendBatch(e []events.Event) {}

func verifyAssets(r *reporter, assets map[string]asset) {
	if assets == nil {
		return // this is fine
	}

	for k, v := range assets {
		if n, failed := num.UintFromString(v.Quantum, 10); failed || n.IsNegative() || n.IsZero() {
			r.Err("app_state.assets[%s].quantum not a valid positive number: %s", k, v.Quantum)
		}

		switch {
		case v.Source == nil:
			r.Err("app_state.assets[%s].source is missing", k)
		case v.Source.BuiltInAsset != nil && v.Source.ERC20 != nil:
			r.Err("app_state.assets[%s].source cannot be both builtin or ERC20", k)
		case v.Source.BuiltInAsset != nil:
			if _, failed := num.UintFromString(v.Source.BuiltInAsset.MaxFaucetAmountMint, 10); failed {
				r.Err("app_state.assets[%s].source.builtin_asset.max_faucet_amount_mint is not a valid number: %s",
					k, v.Source.BuiltInAsset.MaxFaucetAmountMint)
			}
		case v.Source.ERC20 != nil:
			if !isValidParty(k) {
				r.Err("app_state.assets contains an non valid asset id, `%v`", k)
			}
			if len(v.Source.ERC20.ContractAddress) <= 0 {
				r.Err("app_state.assets[%s] contains an empty contract address", k)
			} else if !isValidEthereumAddress(v.Source.ERC20.ContractAddress) {
				r.Err("app_state.assets[%s] contains an invalid ethereum contract address %s", k, v.Source.ERC20.ContractAddress)
			}
		default:
			r.Err("app_state.assets[%s].source must be either builtin or ERC20", k)
		}
	}
}

func verifyValidators(r *reporter, validators map[string]validator) {
	if validators == nil || len(validators) <= 0 {
		r.Warn("app_state.validators is missing or empty")
		return
	}

	for key, v := range validators {
		switch {
		case len(key) <= 0:
			r.Err("app_state.validators contains an empty key")
		case !isValidCometBFTKey(key):
			r.Err("app_state.validators contains an invalid CometBFT public key, `%v`", key)
		case key != v.TMPubKey:
			r.Err("app_state.validator[%v] hash mismatched CometBFT public key, `%v`", key, v.TMPubKey)
		}

		if !isValidParty(v.ID) {
			r.Err("app_state.validators[%v] has an invalid id, `%v`", key, v.ID)
		}

		if !isValidParty(v.VegaPubKey) {
			r.Err("app_state.validators[%v] has an invalid vega public key, `%v`", key, v.VegaPubKey)
		}

		if v.VegaPubKeyIndex == 0 {
			r.Err("app_state.validators[%v] has an invalid vega public key index, `%v`", key, v.VegaPubKeyIndex)
		}

		if !isValidEthereumAddress(v.EthereumAddress) {
			r.Err("app_state.validators[%v] has an invalid ethereum address, `%v`", key, v.EthereumAddress)
		}
	}
}

func verifyNetworkParameters(r *reporter, nps map[string]string, overwriteParameters []string) {
	if nps == nil {
		r.Err("app_state.network_parameters is missing")
		return
	}

	log := logging.NewTestLogger()

	netp := netparams.New(
		log,
		netparams.NewDefaultConfig(),
		noopBroker{},
	)

	// check for no missing keys
	for k := range netparams.AllKeys {
		if _, ok := nps[k]; !ok {
			val, _ := netp.Get(k)
			r.Warn("app_state.network_parameters missing parameter `%v`, default value will be used `%v`", k, val)
		}
	}

	// check for no unknown keys or invalid values
	for k, v := range nps {
		if _, ok := netparams.AllKeys[k]; !ok {
			r.Err("appstate.network_parameters unknown parameter `%v`", k)
			continue
		}

		err := netp.Validate(k, v)
		if err != nil {
			r.Err("appstate.network_parameters invalid parameter `%v`, %v", k, err)
		}
	}

	// check overwrite parameters are real
	for _, k := range overwriteParameters {
		if _, ok := netparams.AllKeys[k]; !ok {
			r.Err("appstate.network_parameters_checkpoint_overwrite unknown parameter `%v`", k)
			continue
		}
	}
}

func verifyGenesis(r *reporter, bs []byte) string {
	// Unmarshal to get appstate
	g := struct {
		AppState        json.RawMessage `json:"app_state"`
		ConsensusParams struct {
			Block struct {
				TimeIotaMs string `json:"time_iota_ms"`
			} `json:"block"`
		} `json:"consensus_params"`
	}{}

	if err := json.Unmarshal(bs, &g); err != nil {
		r.Err("unable to unmarshal genesis file, %v", err)
		return ""
	}

	if g.ConsensusParams.Block.TimeIotaMs != "1" {
		r.Err("consensus_params.block.time_iota_ms must be 1")
	}

	appstate := &appState{}
	d := json.NewDecoder(bytes.NewBuffer(g.AppState))
	d.DisallowUnknownFields() // This allows us to fail if an appstate field is found which we don't know about

	if err := d.Decode(appstate); err != nil {
		r.Err("unable to unmarshal app_state in genesis file, %v", err)
		return ""
	}

	if appstate.NetworkLimits == nil {
		r.Warn("app_state.network_limits are missing, default values will be used")
	}

	switch {
	case appstate.Network == nil:
		r.Err("app_state.network is missing")
	case appstate.Network.ReplayAttackThreshold == nil:
		r.Err("app_state.network.replay_attach_threshold is missing")
	case *appstate.Network.ReplayAttackThreshold < 0:
		r.Err("app_state.network.replace_attach_threshold can't be < 0")
	}

	verifyNetworkParameters(r, appstate.NetworkParameters, appstate.NetworkParametersCPOverwrite)
	verifyValidators(r, appstate.Validators)
	verifyAssets(r, appstate.Assets)

	out, _ := vgjson.Prettify(g)
	return string(out)
}
