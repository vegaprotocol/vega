// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ratelimit

import (
	"time"

	"code.vegaprotocol.io/vega/datanode/config/encoding"
)

type Config struct {
	Enabled        bool              `description:"Enable rate limit of API requests per IP address. Based on a 'token bucket' algorithm"                                              long:"enabled"`
	TrustedProxies []string          `description:"specify a trusted proxy for forwarded requests"                                                                                     long:"trusted-proxy"`
	Rate           float64           `description:"Refill rate of token bucket; maximum average request rate"                                                                          long:"rate"`
	Burst          int               `description:"Size of token bucket; maximum number of requests in short time window"                                                              long:"burst"`
	TTL            encoding.Duration `description:"Time after which inactive token buckets are reset"                                                                                  long:"ttl"`
	BanFor         encoding.Duration `description:"If IP continues to make requests after passing rate limit threshold, ban for this duration. Setting to 0 seconds disables banning." long:"banfor"`
}

func NewDefaultConfig() Config {
	return Config{
		Enabled:        true,
		TrustedProxies: []string{"127.0.0.1"},
		Rate:           20,
		Burst:          100,
		TTL:            encoding.Duration{Duration: time.Hour},
		BanFor:         encoding.Duration{Duration: 10 * time.Minute},
	}
}
