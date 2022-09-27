// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package ethereum

import (
	"time"

	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	defaultDurationBetweenTwoRetry = 20 * time.Second
)

type Config struct {
	// Level specifies the logging level of the Ethereum implementation of the
	// Event Forwarder.
	Level                  encoding.LogLevel `long:"log-level"`
	PollEventRetryDuration encoding.Duration
}

func NewDefaultConfig() Config {
	return Config{
		Level:                  encoding.LogLevel{Level: logging.InfoLevel},
		PollEventRetryDuration: encoding.Duration{Duration: defaultDurationBetweenTwoRetry},
	}
}
