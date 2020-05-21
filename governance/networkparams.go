package governance

import "time"

const (
	day  = 24 * time.Hour // day here is 24 hours
	year = 365 * day      // year here is 365 days (ignoring leap years)
)

const (
	// DefaultMinClose is harcoded minimum voting close offset duration
	//(relative to the time proposal is received from the chain)
	DefaultMinClose = 2 * day
	// DefaultMaxClose is harcoded maximum voting close offset duration
	DefaultMaxClose = 1 * year
	// DefaultMinEnact is harcoded minimum enactment offset duration
	DefaultMinEnact = 2 * day // must be >= minClose
	// DefaultMaxEnact is harcoded maximum enactment offset duration
	DefaultMaxEnact = 1 * year
	// DefaultMinParticipationStake is hardcoded minimum participation stake percent
	DefaultMinParticipationStake = 1
)

// NetworkParameters stores governance network parameters
type NetworkParameters struct {
	minClose              time.Duration
	maxClose              time.Duration
	minEnact              time.Duration
	maxEnact              time.Duration
	minParticipationStake uint64
}

// DefaultNetworkParameters returns default, hardcoded, network parameters
func DefaultNetworkParameters() *NetworkParameters {
	result := &NetworkParameters{
		minClose:              DefaultMinClose,
		maxClose:              DefaultMaxClose,
		minEnact:              DefaultMinEnact,
		maxEnact:              DefaultMaxEnact,
		minParticipationStake: DefaultMinParticipationStake,
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
