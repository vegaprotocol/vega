package ratelimit

type Config struct {
	// How many requests
	Requests int `long:"requests" description:" "`

	// In the last `PerNBlocks` blocks
	PerNBlocks int `long:"per-n-blocks" description:" "`
}

// NewDefaultConfig allows 500 requests in the last 10 blocks.
func NewDefaultConfig() Config {
	return Config{
		Requests:   500,
		PerNBlocks: 10,
	}
}
