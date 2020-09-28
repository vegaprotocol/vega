package ratelimit

type Config struct {
	// How many requests
	Requests int

	// In the last `PerNBlocks` blocks
	PerNBlocks int
}

// NewDefaultConfig allows 500 requests in the last 10 blocks.
func NewDefaultConfig() Config {
	return Config{
		Requests:   500,
		PerNBlocks: 10,
	}
}
