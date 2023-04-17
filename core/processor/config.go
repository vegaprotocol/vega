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

package processor

import (
	"code.vegaprotocol.io/vega/core/processor/ratelimit"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "processor"
)

// Config represent the configuration of the processor package.
type Config struct {
	Level               encoding.LogLevel `long:"log-level"`
	LogOrderSubmitDebug encoding.Bool     `long:"log-order-submit-debug"`
	LogOrderAmendDebug  encoding.Bool     `long:"log-order-amend-debug"`
	LogOrderCancelDebug encoding.Bool     `long:"log-order-cancel-debug"`
	Ratelimit           ratelimit.Config  `group:"Ratelimit" namespace:"ratelimit"`
	KeepCheckpointsMax  uint              `long:"keep-checkpoints-max"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:               encoding.LogLevel{Level: logging.InfoLevel},
		LogOrderSubmitDebug: true,
		Ratelimit:           ratelimit.NewDefaultConfig(),
		KeepCheckpointsMax:  20,
	}
}
