package verify

import (
	"encoding/json"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"

	"github.com/jessevdk/go-flags"
)

type GenesisCmd struct {
	Help bool `short:"h" long:"help" description:"Show this help message"`
}

func (opts *GenesisCmd) Execute(params []string) error {
	if opts.Help {
		return &flags.Error{
			Type:    flags.ErrHelp,
			Message: "vega verify genesis subcommand help",
		}
	}
	return verifier(params, verifyGenesis)
}

type noopBroker struct{}

func (n noopBroker) Send(e events.Event) {}

func verifyGenesis(r *reporter, bs []byte) string {
	var g = &struct {
		AppState *struct {
			Network *struct {
				ReplayAttackThreshold *int `json:"replay_attack_threshold"`
			} `json:"network"`
			NetworkParameters map[string]string `json:"network_parameters"`
			Validators        map[string]string `json:"validators"`
			Assets            *struct {
				ERC20 map[string]types.ERC20 `json:"ERC20"`
			} `json:"assets"`
		} `json:"app_state"`
	}{}

	err := json.Unmarshal(bs, g)
	if err != nil {
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

	if g.AppState.NetworkParameters == nil {
		r.Err("app_state.network_parameters is missing")
	} else {
		log := logging.NewTestLogger()

		netp := netparams.New(
			log,
			netparams.NewDefaultConfig(),
			noopBroker{},
		)
		// first check for no missing keys
		for k := range netparams.AllKeys {
			if _, ok := g.AppState.NetworkParameters[k]; !ok {
				val, _ := netp.Get(k)
				r.Warn("missing network parameter `%v`, default value will be used `%v`", k, val)
			}
		}

		// and now for no unknown keys or invalid values
		for k, v := range g.AppState.NetworkParameters {
			if _, ok := netparams.AllKeys[k]; !ok {
				r.Err("unknow network parameter `%v`", k)
				continue
			}
			err := netp.Validate(k, v)
			if err != nil {
				r.Err("invalid parameter `%v`, %v", k, err)
			}
		}
	}

	if g.AppState.Validators == nil || len(g.AppState.Validators) <= 0 {
		r.Warn("app_state.validators is missing or empty")
	} else {
		for k, v := range g.AppState.Validators {
			if len(k) <= 0 {
				r.Err("app_state.validators contains an empty key")
			} else if !isValidTMKey(k) {
				r.Err("app_state.validators contains an non valid TM public key, `%v`", k)
			}
			if len(v) <= 0 {
				r.Err("app_state.validators contains an empty value for key `%v`", k)
			} else if !isValidParty(v) {
				r.Err("app_state.validators contains an non valid vega public key, `%v`", v)
			}
		}
	}

	if g.AppState.Assets == nil {
		r.Warn("no assets specified as part of the genesis")
	} else {
		for k, v := range g.AppState.Assets.ERC20 {
			if len(k) <= 0 {
				r.Err("app_state.assets contains an empty key")
			} else if !isValidParty(k) {
				r.Err("app_state.assets contains an non valid asset id, `%v`", k)
			}
			if len(v.ContractAddress) <= 0 {
				r.Err("app_state.assets contains an empty contract address for key `%v`", k)
			} else if !isValidEthereumAddress(v.ContractAddress) {
				r.Err("app_state.assets contains an non valid ethereum contract address `%v`", v.ContractAddress)
			}

		}
	}

	out, _ := json.MarshalIndent(g, "  ", "  ")
	return string(out)
}
