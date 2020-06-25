package governance

import (
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

const (
	day  = 24 * time.Hour // day here is 24 hours
	year = 365 * day      // year here is 365 days (ignoring leap years)
)

// These Governance parameters are fixed, unless overridden by ldflags for test purposes.
var (
	MinClose              = ""
	MaxClose              = ""
	MinEnact              = ""
	MaxEnact              = ""
	RequiredParticipation = ""
	RequiredMajority      = ""
	MinProposerBalance    = ""
	MinVoterBalance       = ""
)

const (
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

// FutureOracle stores future product oracle configuration
type FutureOracle struct {
	ContractID string
	Event      string
	Value      uint64
}

// NetworkParameters stores network parameters per proposal type
type NetworkParameters struct {
	NewMarkets          ProposalParameters
	MarginConfiguration ScalingFactors
	FutureOracle        FutureOracle
	InitialMarkPrice    uint64
}

// DefaultNetworkParameters returns default, hardcoded, network parameters
func DefaultNetworkParameters(log *logging.Logger) *NetworkParameters {
	return &NetworkParameters{
		NewMarkets:          defaultNewMarketParameters(log),
		MarginConfiguration: defaultMarginConfiguration(),
		FutureOracle:        defaultFutureOracle(),
		InitialMarkPrice:    1,
	}
}

func defaultMarginConfiguration() ScalingFactors {
	return ScalingFactors{
		SearchLevel:       1.1,
		InitialMargin:     1.2,
		CollateralRelease: 1.4,
	}
}

func defaultFutureOracle() FutureOracle {
	return FutureOracle{
		ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
		Event:      "price_changed",
		Value:      1500000,
	}
}

func defaultNewMarketParameters(log *logging.Logger) ProposalParameters {
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

	if len(MinClose) > 0 {
		result.MinClose, err = time.ParseDuration(MinClose)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MinClose", MinClose),
				logging.Error(err),
			)
		}
	}
	if len(MaxClose) > 0 {
		result.MaxClose, err = time.ParseDuration(MaxClose)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MaxClose", MaxClose),
				logging.Error(err),
			)
		}
	}
	if len(MinEnact) > 0 {
		result.MinEnact, err = time.ParseDuration(MinEnact)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MinEnact", MinEnact),
				logging.Error(err),
			)
		}
	}
	if len(MaxEnact) > 0 {
		result.MaxEnact, err = time.ParseDuration(MaxEnact)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MaxEnact", MaxEnact),
				logging.Error(err),
			)
		}
	}
	if len(RequiredParticipation) > 0 {
		levelValue, err := strconv.ParseFloat(RequiredParticipation, 32)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("RequiredParticipation", RequiredParticipation),
				logging.Error(err),
			)
		}
		if levelValue < 0 {
			log.Fatal(
				"New market network parameter is invalid (negative)",
				logging.String("RequiredParticipation", RequiredParticipation),
			)
		}
		if levelValue > 1 {
			log.Fatal(
				"New market network parameter is invalid (over 1)",
				logging.String("RequiredParticipation", RequiredParticipation),
			)
		}
		result.RequiredParticipation = float32(levelValue)
	}
	if len(RequiredMajority) > 0 {
		levelValue, err := strconv.ParseFloat(RequiredMajority, 32)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("RequiredMajority", RequiredMajority),
				logging.Error(err),
			)
		}
		if levelValue < 0.5 {
			log.Fatal(
				"New market network parameter is invalid (less than 0.5)",
				logging.String("RequiredMajority", RequiredMajority),
			)
		}
		if levelValue > 1 {
			log.Fatal(
				"New market network parameter is invalid (over 1)",
				logging.String("RequiredMajority", RequiredMajority),
			)
		}
		result.RequiredMajority = float32(levelValue)
	}
	if len(MinProposerBalance) > 0 {
		levelValue, err := strconv.ParseFloat(MinProposerBalance, 32)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MinProposerBalance", MinProposerBalance),
				logging.Error(err),
			)
		}
		if levelValue <= 0 {
			log.Fatal(
				"New market network parameter is invalid (less or equal than 0)",
				logging.String("MinProposerBalance", MinProposerBalance),
			)
		}
		if levelValue > 1 {
			log.Fatal(
				"New market network parameter is invalid (over 1)",
				logging.String("MinProposerBalance", MinProposerBalance),
			)
		}
		result.MinProposerBalance = float32(levelValue)
	}
	if len(MinVoterBalance) > 0 {
		levelValue, err := strconv.ParseFloat(MinVoterBalance, 32)
		if err != nil {
			log.Fatal(
				"Failed to parse new market network parameter",
				logging.String("MinVoterBalance", MinVoterBalance),
				logging.Error(err),
			)
		}
		if levelValue <= 0 {
			log.Fatal(
				"New market network parameter is invalid (less or equal than 0)",
				logging.String("MinVoterBalance", MinVoterBalance),
			)
		}
		if levelValue > 1 {
			log.Fatal(
				"New market network parameter is invalid (over 1)",
				logging.String("MinVoterBalance", MinVoterBalance),
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
