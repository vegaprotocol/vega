package governance

import (
	"fmt"
	"strconv"
	"time"
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

// NetworkParameters stores governance network parameters
type NetworkParameters struct {
	minClose              time.Duration
	maxClose              time.Duration
	minEnact              time.Duration
	maxEnact              time.Duration
	requiredParticipation float32
	requiredMajority      float32
	minProposerBalance    float32
	minVoterBalance       float32
}

// DefaultNetworkParameters returns default, hardcoded, network parameters
func DefaultNetworkParameters() *NetworkParameters {
	var err error
	result := &NetworkParameters{
		minClose:              defaultMinClose,
		maxClose:              defaultMaxClose,
		minEnact:              defaultMinEnact,
		maxEnact:              defaultMaxEnact,
		requiredParticipation: defaultRequiredParticipation,
		requiredMajority:      defaultRequiredMajority,
		minProposerBalance:    defaultMinProposerBalance,
		minVoterBalance:       defaultMinVoterBalance,
	}
	if len(MinClose) > 0 {
		result.minClose, err = time.ParseDuration(MinClose)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse time duration, %s: %s", MinClose, err.Error()))
		}
	}
	if len(MaxClose) > 0 {
		result.maxClose, err = time.ParseDuration(MaxClose)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse time duration, %s: %s", MaxClose, err.Error()))
		}
	}
	if len(MinEnact) > 0 {
		result.minEnact, err = time.ParseDuration(MinEnact)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse time duration, %s: %s", MinEnact, err.Error()))
		}
	}
	if len(MaxEnact) > 0 {
		result.maxEnact, err = time.ParseDuration(MaxEnact)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse time duration, %s: %s", MaxEnact, err.Error()))
		}
	}
	if len(RequiredParticipation) > 0 {
		levelValue, err := strconv.ParseFloat(RequiredParticipation, 32)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse RequiredParticipation, %s: %s", RequiredParticipation, err.Error()))
		}
		if levelValue < 0 {
			panic(fmt.Sprintf("Invalid RequiredParticipation (negative): %s", RequiredParticipation))
		}
		if levelValue > 1 {
			panic(fmt.Sprintf("Invalid RequiredParticipation (over 1): %s", RequiredParticipation))
		}
		result.requiredParticipation = float32(levelValue)
	}
	if len(RequiredMajority) > 0 {
		levelValue, err := strconv.ParseFloat(RequiredMajority, 32)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse RequiredMajority, %s: %s", RequiredMajority, err.Error()))
		}
		if levelValue < 0.5 {
			panic(fmt.Sprintf("Invalid RequiredMajority (less than 0.5): %s", RequiredMajority))
		}
		if levelValue > 1 {
			panic(fmt.Sprintf("Invalid RequiredMajority (over 1): %s", RequiredMajority))
		}
		result.requiredMajority = float32(levelValue)
	}
	if len(MinProposerBalance) > 0 {
		levelValue, err := strconv.ParseFloat(MinProposerBalance, 32)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse MinProposerBalance, %s: %s", MinProposerBalance, err.Error()))
		}
		if levelValue <= 0 {
			panic(fmt.Sprintf("Invalid MinProposingBalance (less or equal than 0): %s", MinProposerBalance))
		}
		if levelValue > 1 {
			panic(fmt.Sprintf("Invalid MinProposingBalance (over 1): %s", MinProposerBalance))
		}
		result.minProposerBalance = float32(levelValue)
	}
	if len(MinVoterBalance) > 0 {
		levelValue, err := strconv.ParseFloat(MinVoterBalance, 32)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse MinVoterBalance, %s: %s", MinVoterBalance, err.Error()))
		}
		if levelValue <= 0 {
			panic(fmt.Sprintf("Invalid MinVoterBalance (less or equal than 0): %s", MinVoterBalance))
		}
		if levelValue > 1 {
			panic(fmt.Sprintf("Invalid MinVoterBalance (over 1): %s", MinVoterBalance))
		}
		result.minVoterBalance = float32(levelValue)
	}

	result.maxClose = max(result.maxClose, result.minClose) // maxClose must be >= minClose
	result.minEnact = max(result.minEnact, result.minClose) // minEnact must be >= minClose
	result.maxEnact = max(result.maxEnact, result.minEnact) // maxEnact must be >= minEnact
	return result
}

func max(left, right time.Duration) time.Duration {
	if left > right {
		return left
	}
	return right
}
