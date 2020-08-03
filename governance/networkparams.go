package governance

import (
	"log"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

const (
	day  = 24 * time.Hour // day here is 24 hours
	year = 365 * day      // year here is 365 days (ignoring leap years)

	// defaultMinClose is the hardcoded minimum voting close offset duration
	// (relative to the time proposal is received from the chain)
	defaultMinClose = 2 * day
	// defaultMaxClose is the hardcoded maximum voting close offset duration
	defaultMaxClose = 1 * year
	// defaultMinEnact is the hardcoded minimum enactment offset duration
	defaultMinEnact = 2 * day // must be >= minClose
	// defaultMaxEnact is the hardcoded maximum enactment offset duration
	defaultMaxEnact = 1 * year
	// defaultRequiredParticipation is hardcoded participation level required for any proposal to pass (from `0` to `1`)
	defaultRequiredParticipation = 0.00001
	// defaultRequiredMajority is hardcoded majority level required for any proposal to pass (from `0.5` to `1`)
	defaultRequiredMajority = 0.66
	// defaultProposerBalance is hardcoded minimum balance required for a party
	// to be able to submit a new proposal (greater than `0` to `1`)
	defaultMinProposerBalance = 0.00001
	// defaultMinVoterBalance is hardcoded minimum balance required for a party
	// to be able to cast a vote (greater than `0` to `1`).
	defaultMinVoterBalance = 0.00001

	defaultMakerFee          = "0.00025"
	defaultInfrastructureFee = "0.0005"
	defaultLiquidityFee      = "0.001"

	defaultSearchLevel       = "1.1"
	defaultInitialMargin     = "1.2"
	defaultCollateralRelease = "1.4"
)

// ProposalParameters stores proposal specific parameters
type ProposalParameters struct {
	MinClose              time.Duration
	MaxClose              time.Duration
	MinEnact              time.Duration
	MaxEnact              time.Duration
	RequiredParticipation float32
	RequiredMajority      float32
	MinProposerBalance    float32
	MinVoterBalance       float32
}

// ScalingFactors stores scaling factors for all markets created via governance
type ScalingFactors struct {
	SearchLevel       float64
	InitialMargin     float64
	CollateralRelease float64
}

// FeeFactos store factors for the the which are not specified through a proposal
// and are global to all markets
type FeeFactors struct {
	MakerFee          string
	InfrastructureFee string
	LiquidityFee      string
}

// FutureOracle stores future product oracle configuration
type FutureOracle struct {
	ContractID string
	Event      string
	Value      uint64
}

// NetworkParameters stores network parameters per proposal type
type NetworkParameters struct {
	Proposals           ProposalParameters
	MarginConfiguration ScalingFactors
	FutureOracle        FutureOracle
	InitialMarkPrice    uint64
	FeeFactors          FeeFactors
}

// DefaultNetworkParameters returns default, hardcoded, network parameters
func DefaultNetworkParameters(log *logging.Logger) *NetworkParameters {
	gstate := DefaultGenesisState()
	return NetworkParametersFromGenesisState(log, gstate)
}

// NetworkParametersFromGenesisState returns network parameter loaded from the
// genesis state
func NetworkParametersFromGenesisState(log *logging.Logger, gstate GenesisState) *NetworkParameters {
	return &NetworkParameters{
		Proposals:           defaultProposalParameters(log, gstate),
		MarginConfiguration: defaultMarginConfiguration(gstate),
		FutureOracle:        defaultFutureOracle(),
		InitialMarkPrice:    1,
		FeeFactors:          defaultFeeFactors(gstate),
	}
}

func defaultFeeFactors(gstate GenesisState) FeeFactors {
	return FeeFactors{
		MakerFee:          gstate.MakerFee,
		InfrastructureFee: gstate.InfrastructureFee,
		LiquidityFee:      gstate.LiquidityFee,
	}
}

func defaultMarginConfiguration(gstate GenesisState) ScalingFactors {
	search, err := strconv.ParseFloat(gstate.SearchLevel, 32)
	if err != nil {
		log.Fatal(
			"Failed to parse margins search level",
			logging.String("SearchLevel", gstate.SearchLevel),
			logging.Error(err))
	}
	initial, err := strconv.ParseFloat(gstate.InitialMargin, 32)
	if err != nil {
		log.Fatal(
			"Failed to parse margins search level",
			logging.String("SearchLevel", gstate.InitialMargin),
			logging.Error(err))
	}
	release, err := strconv.ParseFloat(gstate.CollateralRelease, 32)
	if err != nil {
		log.Fatal(
			"Failed to parse margins search level",
			logging.String("SearchLevel", gstate.CollateralRelease),
			logging.Error(err))
	}

	return ScalingFactors{
		SearchLevel:       search,
		InitialMargin:     initial,
		CollateralRelease: release,
	}
}

func defaultFutureOracle() FutureOracle {
	return FutureOracle{
		ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
		Event:      "price_changed",
		Value:      1500000,
	}
}

func defaultProposalParameters(log *logging.Logger, gstate GenesisState) ProposalParameters {
	var err error
	result := ProposalParameters{
		MinClose:              defaultMinClose,
		MaxClose:              defaultMaxClose,
		MinEnact:              defaultMinEnact,
		MaxEnact:              defaultMaxEnact,
		RequiredParticipation: defaultRequiredParticipation,
		RequiredMajority:      defaultRequiredMajority,
		MinProposerBalance:    defaultMinProposerBalance,
		MinVoterBalance:       defaultMinVoterBalance,
	}

	if len(gstate.MinClose) > 0 {
		result.MinClose, err = time.ParseDuration(gstate.MinClose)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MinClose", gstate.MinClose),
				logging.Error(err),
			)
		}
	}
	if len(gstate.MaxClose) > 0 {
		result.MaxClose, err = time.ParseDuration(gstate.MaxClose)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MaxClose", gstate.MaxClose),
				logging.Error(err),
			)
		}
	}
	if len(gstate.MinEnact) > 0 {
		result.MinEnact, err = time.ParseDuration(gstate.MinEnact)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MinEnact", gstate.MinEnact),
				logging.Error(err),
			)
		}
	}
	if len(gstate.MaxEnact) > 0 {
		result.MaxEnact, err = time.ParseDuration(gstate.MaxEnact)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MaxEnact", gstate.MaxEnact),
				logging.Error(err),
			)
		}
	}
	if len(gstate.RequiredParticipation) > 0 {
		levelValue, err := strconv.ParseFloat(gstate.RequiredParticipation, 32)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("RequiredParticipation", gstate.RequiredParticipation),
				logging.Error(err),
			)
		}
		if levelValue < 0 {
			log.Fatal(
				"New market network parameter is invalid (negative)",
				logging.String("RequiredParticipation", gstate.RequiredParticipation),
			)
		}
		if levelValue > 1 {
			log.Fatal(
				"New market network parameter is invalid (over 1)",
				logging.String("RequiredParticipation", gstate.RequiredParticipation),
			)
		}
		result.RequiredParticipation = float32(levelValue)
	}
	if len(gstate.RequiredMajority) > 0 {
		levelValue, err := strconv.ParseFloat(gstate.RequiredMajority, 32)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("RequiredMajority", gstate.RequiredMajority),
				logging.Error(err),
			)
		}
		if levelValue < 0.5 {
			log.Fatal(
				"New market network parameter is invalid (less than 0.5)",
				logging.String("RequiredMajority", gstate.RequiredMajority),
			)
		}
		if levelValue > 1 {
			log.Fatal(
				"New market network parameter is invalid (over 1)",
				logging.String("RequiredMajority", gstate.RequiredMajority),
			)
		}
		result.RequiredMajority = float32(levelValue)
	}
	if len(gstate.MinProposerBalance) > 0 {
		levelValue, err := strconv.ParseFloat(gstate.MinProposerBalance, 32)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MinProposerBalance", gstate.MinProposerBalance),
				logging.Error(err),
			)
		}
		if levelValue <= 0 {
			log.Fatal(
				"New market network parameter is invalid (less or equal than 0)",
				logging.String("MinProposerBalance", gstate.MinProposerBalance),
			)
		}
		if levelValue > 1 {
			log.Fatal(
				"New market network parameter is invalid (over 1)",
				logging.String("MinProposerBalance", gstate.MinProposerBalance),
			)
		}
		result.MinProposerBalance = float32(levelValue)
	}
	if len(gstate.MinVoterBalance) > 0 {
		levelValue, err := strconv.ParseFloat(gstate.MinVoterBalance, 32)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MinVoterBalance", gstate.MinVoterBalance),
				logging.Error(err),
			)
		}
		if levelValue <= 0 {
			log.Fatal(
				"New market network parameter is invalid (less or equal than 0)",
				logging.String("MinVoterBalance", gstate.MinVoterBalance),
			)
		}
		if levelValue > 1 {
			log.Fatal(
				"New market network parameter is invalid (over 1)",
				logging.String("MinVoterBalance", gstate.MinVoterBalance),
			)
		}
		result.MinVoterBalance = float32(levelValue)
	}

	if result.MaxClose < result.MinClose {
		log.Fatal(
			"New market MaxClose network parameter is less than MinClose",
			logging.String("MaxClose", result.MaxClose.String()),
			logging.String("MinClose", result.MinClose.String()),
		)
	}
	if result.MaxEnact < result.MinEnact {
		log.Fatal(
			"New market MaxEnact network parameter is less than MinEnact",
			logging.String("MaxEnact", result.MaxEnact.String()),
			logging.String("MinEnact", result.MinEnact.String()),
		)
	}
	if result.MinEnact < result.MinClose {
		log.Fatal(
			"New market MinEnact network parameter is less than MinClose",
			logging.String("MinEnact", result.MinEnact.String()),
			logging.String("MinClose", result.MinClose.String()),
		)
	}
	return result
}
