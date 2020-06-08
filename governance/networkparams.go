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
	MinParticipationStake = ""
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
	// defaultMinParticipationStake is hardcoded minimum participation stake fraction (from `0` to `1`)
	defaultMinParticipationStake = 0.01
)

// NetworkParameters stores governance network parameters
type NetworkParameters struct {
	minClose              time.Duration
	maxClose              time.Duration
	minEnact              time.Duration
	maxEnact              time.Duration
	minParticipationStake float32
}

// DefaultNetworkParameters returns default, hardcoded, network parameters
func DefaultNetworkParameters() *NetworkParameters {
	var err error
	result := &NetworkParameters{
		minClose:              defaultMinClose,
		maxClose:              defaultMaxClose,
		minEnact:              defaultMinEnact,
		maxEnact:              defaultMaxEnact,
		minParticipationStake: defaultMinParticipationStake,
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
	if len(MinParticipationStake) > 0 {
		stakeValue, err := strconv.ParseFloat(MinParticipationStake, 32)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse float value, %s: %s", MinParticipationStake, err.Error()))
		}
		if stakeValue < 0 {
			panic(fmt.Sprintf("Invalid MinParticipationStake (negative): %s", MinParticipationStake))
		}
		if stakeValue > 1 {
			panic(fmt.Sprintf("Invalid MinParticipationStake (over 1): %s", MinParticipationStake))
		}
		result.minParticipationStake = float32(stakeValue)
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
