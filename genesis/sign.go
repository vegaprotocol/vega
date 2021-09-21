package genesis

import (
	"encoding/json"
	"sort"
	"time"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/vega/netparams"
	tmconfig "github.com/tendermint/tendermint/config"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	signedNetworkParameters = []string{
		netparams.BlockchainsEthereumConfig,
		netparams.GovernanceProposalAssetMinEnact,
		netparams.GovernanceProposalMarketMinEnact,
		netparams.GovernanceProposalUpdateMarketRequiredMajority,
		netparams.GovernanceProposalUpdateMarketRequiredParticipation,
		netparams.NetworkCheckpointNetworkEOLDate,
		netparams.ValidatorsEpochLength,
	}
)

func GetSignedParameters(genesisState *GenesisState) (*SignedParameters, error) {
	sps := &SignedParameters{}
	extractNetworkParameters(genesisState, sps)
	extractNetworkLimits(genesisState, sps)
	extractAssets(genesisState, sps)

	return sps, nil
}

func GetLocalGenesisState(path string) (*tmtypes.GenesisDoc, *GenesisState, error) {
	tmConfig := tmconfig.DefaultConfig()
	tmConfig.SetRoot(path)
	genesisFilePath := tmConfig.GenesisFile()

	data, err := vgfs.ReadFile(genesisFilePath)
	if err != nil {
		return nil, nil, err
	}

	return GenesisFromJSON(data)
}

func GenesisFromJSON(rawGenesisDoc []byte) (*tmtypes.GenesisDoc, *GenesisState, error) {
	genesisDoc, err := tmtypes.GenesisDocFromJSON(rawGenesisDoc)
	if err != nil {
		return nil, nil, err
	}

	genesisState := &GenesisState{}
	err = json.Unmarshal(genesisDoc.AppState, genesisState)
	if err != nil {
		return nil, nil, err
	}
	return genesisDoc, genesisState, nil
}

func extractAssets(genesisState *GenesisState, sps *SignedParameters) {
	ads := make([]assetDetails, len(genesisState.Assets))
	i := 0
	for _, details := range genesisState.Assets {
		src := &source{}
		if details.Source.BuiltinAsset != nil {
			src.BuiltinAsset = &builtinAsset{
				MaxFaucetAmountMint: details.Source.BuiltinAsset.MaxFaucetAmountMint,
			}
		} else if details.Source.Erc20 != nil {
			src.Erc20 = &erc20{
				ContractAddress: details.Source.Erc20.ContractAddress,
			}
		}

		ads[i] = assetDetails{
			Name:        details.Name,
			Symbol:      details.Symbol,
			TotalSupply: details.TotalSupply,
			Decimals:    details.Decimals,
			MinLpStake:  details.MinLpStake,
			Source:      src,
		}
		i += 1
	}
	sps.Assets = ads
}

func extractNetworkParameters(genesisState *GenesisState, sps *SignedParameters) {
	for _, name := range signedNetworkParameters {
		sps.NetworkParameters = append(sps.NetworkParameters, networkParams{
			Name:  name,
			Value: genesisState.NetParams[name],
		})
	}
}

func extractNetworkLimits(genesisState *GenesisState, sps *SignedParameters) {
	sps.Limits = networkLimits{
		ProposeMarketEnabled:     genesisState.Limits.ProposeMarketEnabled,
		ProposeAssetEnabled:      genesisState.Limits.ProposeAssetEnabled,
		ProposeMarketEnabledFrom: genesisState.Limits.ProposeMarketEnabledFrom,
		ProposeAssetEnabledFrom:  genesisState.Limits.ProposeAssetEnabledFrom,
	}
}

// SignedParameters represents the data to be signed.
// DO NOT CHANGE THE ORDER OF THE PARAMETERS.
// Changing the order will produce a different signature.
type SignedParameters struct {
	Assets            []assetDetails  `json:"assets"`
	NetworkParameters []networkParams `json:"network_parameters"`
	Limits            networkLimits   `json:"network_limits"`
}

func (p SignedParameters) MarshalJSON() ([]byte, error) {
	type alias SignedParameters

	sort.SliceStable(p.NetworkParameters, func(i, j int) bool {
		return p.NetworkParameters[i].Name < p.NetworkParameters[j].Name
	})

	sort.SliceStable(p.Assets, func(i, j int) bool {
		return p.Assets[i].Name < p.Assets[j].Name
	})

	return json.Marshal(alias(p))
}

// DO NOT CHANGE THE ORDER OF THE PARAMETERS.
// Changing the order will produce a different signature.
type networkParams struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DO NOT CHANGE THE ORDER OF THE PARAMETERS.
// Changing the order will produce a different signature.
type networkLimits struct {
	ProposeMarketEnabled     bool       `json:"propose_market_enabled"`
	ProposeAssetEnabled      bool       `json:"propose_asset_enabled"`
	ProposeMarketEnabledFrom *time.Time `json:"propose_market_enabled_from,omitempty"`
	ProposeAssetEnabledFrom  *time.Time `json:"propose_asset_enabled_from,omitempty"`
}

// DO NOT CHANGE THE ORDER OF THE PARAMETERS.
// Changing the order will produce a different signature.
type assetDetails struct {
	Name        string  `json:"name"`
	Symbol      string  `json:"symbol"`
	TotalSupply string  `json:"total_supply"`
	Decimals    uint64  `json:"decimals"`
	MinLpStake  string  `json:"min_lp_stake"`
	Source      *source `json:"source"`
}

// DO NOT CHANGE THE ORDER OF THE PARAMETERS.
// Changing the order will produce a different signature.
type source struct {
	BuiltinAsset *builtinAsset `json:"builtin_asset,omitempty"`
	Erc20        *erc20        `json:"erc20,omitempty"`
}

// DO NOT CHANGE THE ORDER OF THE PARAMETERS.
// Changing the order will produce a different signature.
type builtinAsset struct {
	MaxFaucetAmountMint string `json:"max_faucet_amount_mint"`
}

// DO NOT CHANGE THE ORDER OF THE PARAMETERS.
// Changing the order will produce a different signature.
type erc20 struct {
	ContractAddress string `json:"contract_address"`
}
