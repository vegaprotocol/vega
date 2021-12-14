package verify

import (
	"encoding/json"

	types "code.vegaprotocol.io/protos/vega"
	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
)

type validator struct {
	ID              string `json:"id"`
	VegaPubKey      string `json:"vega_pub_key"`
	VegaPubKeyIndex int    `json:"vega_pub_key_index"`
	EthereumAddress string `json:"ethereum_address"`
	TMPubKey        string `json:"tm_pub_key"`
	InfoURL         string `json:"info_url"`
	Country         string `json:"country"`
	Name            string `json:"name"`
	AvatarURL       string `json:"avatar_url"`
}

type genesis struct {
	AppState *struct {
		Network *struct {
			ReplayAttackThreshold *int `json:"replay_attack_threshold"`
		} `json:"network"`
		NetworkParameters map[string]string    `json:"network_parameters"`
		Validators        map[string]validator `json:"validators,omitempty"`
		Assets            *struct {
			ERC20 map[string]types.ERC20 `json:"ERC20"`
		} `json:"assets"`
	} `json:"app_state"`
}

type GenesisCmd struct{}

func (opts *GenesisCmd) Execute(params []string) error {
	return verifier(params, verifyGenesis)
}

type noopBroker struct{}

func (n noopBroker) Send(e events.Event) {}

func (noopBroker) SendBatch(e []events.Event) {}

func verifyValidators(r *reporter, validators map[string]validator) {
	if validators == nil || len(validators) <= 0 {
		r.Warn("app_state.validators is missing or empty")
		return
	}

	for tmkey, v := range validators {

		switch {
		case len(tmkey) <= 0:
			r.Err("app_state.validators contains an empty key")
		case !isValidTMKey(tmkey):
			r.Err("app_state.validators contains an invalid TM public key, `%v`", tmkey)
		case tmkey != v.TMPubKey:
			r.Err("app_state.validator[%v] hash mismatched TM pub key, `%v`", tmkey, v.TMPubKey)
		}

		if !isValidParty(v.ID) {
			r.Err("app_state.validators[%v] has an invalid id, `%v`", tmkey, v.ID)
		}

		if !isValidParty(v.VegaPubKey) {
			r.Err("app_state.validators[%v] has an invalid vega public key, `%v`", tmkey, v.VegaPubKey)
		}

		if v.VegaPubKeyIndex < 1 {
			r.Err("app_state.validators[%v] has an invalid vega public key index, `%v`", v.VegaPubKeyIndex)
		}

		if !isValidEthereumAddress(v.EthereumAddress) {
			r.Err("app_state.validators[%v] has an invalid ethereum address, `%v`", v.EthereumAddress)
		}
	}
}

func verifyNetworkParameters(r *reporter, nps map[string]string) {
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
	// first check for no missing keys
	for k := range netparams.AllKeys {
		if _, ok := nps[k]; !ok {
			val, _ := netp.Get(k)
			r.Warn("missing network parameter `%v`, default value will be used `%v`", k, val)
		}
	}

	// and now for no unknown keys or invalid values
	for k, v := range nps {
		if _, ok := netparams.AllKeys[k]; !ok {
			r.Err("unknown network parameter `%v`", k)
			continue
		}
		err := netp.Validate(k, v)
		if err != nil {
			r.Err("invalid parameter `%v`, %v", k, err)
		}
	}
}

func verifyGenesis(r *reporter, bs []byte) string {
	g := &genesis{}
	if err := json.Unmarshal(bs, &g); err != nil {
		r.Err("unable to unmarshal genesis file, %v", err)
		return ""
	}

	if g.AppState == nil {
		r.Err("app_state is missing")
		return ""
	}

	if g.AppState.Network == nil {
		r.Err("app_state.network is missing")
	} else if g.AppState.Network.ReplayAttackThreshold == nil {
		r.Err("app_state.network.ReplayAttackTreshold is missing")
	} else if *g.AppState.Network.ReplayAttackThreshold < 0 {
		r.Err("app_state.network.ReplayAttackTreshold can't be < 0")
	}

	verifyNetworkParameters(r, g.AppState.NetworkParameters)
	verifyValidators(r, g.AppState.Validators)

	if g.AppState.Assets == nil {
		r.Warn("no assets specified as part of the genesis")
	} else {
		for k, v := range g.AppState.Assets.ERC20 {
			if len(k) <= 0 {
				r.Err("app_state.assets contains an empty key")
			} else if !isValidParty(k) {
				r.Err("app_state.assets contains an invalid asset id, `%v`", k)
			}
			if len(v.ContractAddress) <= 0 {
				r.Err("app_state.assets contains an empty contract address for key `%v`", k)
			} else if !isValidEthereumAddress(v.ContractAddress) {
				r.Err("app_state.assets contains an invalid ethereum contract address `%v`", v.ContractAddress)
			}
		}
	}

	out, _ := vgjson.Prettify(g)
	return string(out)
}
