package governance

import "time"

type networkParameters struct {
	minClose              time.Duration
	maxClose              time.Duration
	minEnact              time.Duration
	maxEnact              time.Duration
	minParticipationStake uint64
}

func defaultNetworkParameters() *networkParameters {

	const day = 24 * time.Hour // day here is 24 hours
	const year = 365 * day     // year here is 365 days (ignoring leap years)

	return &networkParameters{
		minClose:              2 * day,
		maxClose:              1 * year,
		minEnact:              2 * day, // must be >= minClose
		maxEnact:              1 * year,
		minParticipationStake: 1, // percent
	}
}

func readNetworkParameters(cfg Config) *networkParameters {
	result := defaultNetworkParameters()

	if cfg.CloseParameters != nil {
		result.minClose = time.Duration(cfg.CloseParameters.DefaultMinSeconds) * time.Second
		result.maxClose = time.Duration(cfg.CloseParameters.DefaultMaxSeconds) * time.Second
	}
	if cfg.EnactParameters != nil {
		result.minEnact = time.Duration(cfg.EnactParameters.DefaultMinSeconds) * time.Second
		result.maxEnact = time.Duration(cfg.EnactParameters.DefaultMaxSeconds) * time.Second
	}
	if cfg.DefaultMinParticipationStake != 0 { // accepting proposals with no participation makes little sense
		result.minParticipationStake = cfg.DefaultMinParticipationStake
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
