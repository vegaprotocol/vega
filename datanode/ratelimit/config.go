package ratelimit

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
)

type Config struct {
	Enabled bool              `description:"Enable rate limit of API requests per IP address. Based on a 'token bucket' algorithm"                                              long:"enabled"`
	Rate    float64           `description:"Refill rate of token bucket; maximum average request rate"                                                                          long:"rate"`
	Burst   int               `description:"Size of token bucket; maximum number of requests in short time window"                                                              long:"burst"`
	TTL     encoding.Duration `description:"Time after which inactive token buckets are reset"                                                                                  long:"ttl"`
	BanFor  encoding.Duration `description:"If IP continues to make requests after passing rate limit threshold, ban for this duration. Setting to 0 seconds disables banning." long:"banfor"`
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
