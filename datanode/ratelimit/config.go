package ratelimit

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
)

type Config struct {
	Enabled bool              `long:"enabled" description:"Enable rate limit of API requests per IP address. Based on a 'token bucket' algorithm"`
	Rate    float64           `long:"rate" description:"Refill rate of token bucket; maximum average request rate"`
	Burst   int               `long:"burst" description:"Size of token bucket; maximum number of requests in short time window"`
	TTL     encoding.Duration `long:"ttl" description:"Time after which inactive token buckets are reset"`
	BanFor  encoding.Duration `long:"banfor" description:"If IP continues to make requests after passing rate limit threshold, ban for this duration. Setting to 0 seconds disables banning."`
}

func NewDefaultConfig() Config {
	return Config{
		Enabled: true,
		Rate:    20,
		Burst:   100,
		TTL:     encoding.Duration{Duration: time.Hour},
		BanFor:  encoding.Duration{Duration: 10 * time.Minute},
	}
}
