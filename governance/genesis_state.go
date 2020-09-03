package governance

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrNoGovernanceGenesisState = errors.New("no governance genesis state")
)

type GenesisState struct {
	// proposals
	MinClose              string
	MaxClose              string
	MinEnact              string
	MaxEnact              string
	RequiredParticipation string
	RequiredMajority      string
	MinProposerBalance    string
	MinVoterBalance       string
	// fees
	MakerFee          string
	InfrastructureFee string
	LiquidityFee      string
	// Margins
	SearchLevel       string
	InitialMargin     string
	CollateralRelease string
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		MinClose:              defaultMinClose.String(),
		MaxClose:              defaultMaxClose.String(),
		MinEnact:              defaultMinEnact.String(),
		MaxEnact:              defaultMaxEnact.String(),
		RequiredParticipation: fmt.Sprintf("%f", defaultRequiredParticipation),
		RequiredMajority:      fmt.Sprintf("%f", defaultRequiredMajority),
		MinProposerBalance:    fmt.Sprintf("%f", defaultMinProposerBalance),
		MinVoterBalance:       fmt.Sprintf("%f", defaultMinVoterBalance),
		MakerFee:              defaultMakerFee,
		InfrastructureFee:     defaultInfrastructureFee,
		LiquidityFee:          defaultLiquidityFee,
		SearchLevel:           defaultSearchLevel,
		InitialMargin:         defaultInitialMargin,
		CollateralRelease:     defaultCollateralRelease,
	}
}

func LoadGenesisState(bytes []byte) (*GenesisState, error) {
	state := struct {
		Governance *GenesisState `json:"governance"`
	}{}
	err := json.Unmarshal(bytes, &state)
	if err != nil {
		return nil, err
	}
	if state.Governance == nil {
		return nil, ErrNoGovernanceGenesisState
	}
	return state.Governance, nil
}
