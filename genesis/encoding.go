package genesis

import (
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/checkpoint"
	"code.vegaprotocol.io/vega/limits"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/validators"
)

type GenesisState struct {
	Assets             assets.GenesisState             `json:"assets"`
	Validators         validators.GenesisState         `json:"validators"`
	Network            abci.GenesisState               `json:"network"`
	NetParams          netparams.GenesisState          `json:"network_parameters"`
	NetParamsOverwrite netparams.GenesisStateOverwrite `json:"network_parameters_checkpoint_overwrite"`
	Limits             limits.GenesisState             `json:"network_limits"`
	Checkpoint         checkpoint.GenesisState         `json:"checkpoint"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		Limits:     limits.DefaultGenesisState(),
		Assets:     assets.DefaultGenesisState(),
		Validators: validators.DefaultGenesisState(),
		Network:    abci.DefaultGenesis(),
		NetParams:  netparams.DefaultGenesisState(),
		Checkpoint: checkpoint.DefaultGenesisState(),
	}
}
