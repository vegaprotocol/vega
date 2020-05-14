package governance

// default network parameters values
const (
	minCloseSeconds      = 48 * 3600       // 2 days
	maxCloseSeconds      = 365 * 24 * 3600 // 1 year
	minEnactSeconds      = 48 * 3600       // 2 days (must be >= minCloseSeconds)
	maxEnactSeconds      = 365 * 24 * 3600 // 1 year
	participationPercent = 1               // percentage!
)

type networkParameters struct {
	minCloseInSeconds     int64
	maxCloseInSeconds     int64
	minEnactInSeconds     int64
	maxEnactInSeconds     int64
	minParticipationStake uint64
}

func max(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func readNetworkParameters(cfg Config) *networkParameters {
	result := &networkParameters{
		minCloseInSeconds:     minCloseSeconds,
		maxCloseInSeconds:     maxCloseSeconds,
		minEnactInSeconds:     minEnactSeconds,
		maxEnactInSeconds:     maxEnactSeconds,
		minParticipationStake: participationPercent,
	}
	if cfg.CloseParameters != nil {
		result.minCloseInSeconds = cfg.CloseParameters.DefaultMinSeconds
		result.maxCloseInSeconds = cfg.CloseParameters.DefaultMaxSeconds
	}
	if cfg.EnactParameters != nil {
		result.minEnactInSeconds = cfg.EnactParameters.DefaultMinSeconds
		result.minEnactInSeconds = cfg.EnactParameters.DefaultMaxSeconds
	}
	if cfg.DefaultMinParticipationStake != 0 { // accepting proposals with no participation makes little sense
		result.minParticipationStake = cfg.DefaultMinParticipationStake
	}

	result.maxCloseInSeconds = max(result.maxCloseInSeconds, result.minCloseInSeconds) // max close must be >= min close
	result.minEnactInSeconds = max(result.minEnactInSeconds, result.minCloseInSeconds) // min enact must be >= min close
	result.maxEnactInSeconds = max(result.maxEnactInSeconds, result.minEnactInSeconds) // max enact must be >= min enact
	return result
}
