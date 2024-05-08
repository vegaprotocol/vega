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

package processor

import (
	"code.vegaprotocol.io/vega/core/processor/ratelimit"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

const (
	namedLogger = "processor"
)

type Snapshot struct {
	DevEnabled     encoding.Bool
	CrashAtHeight  uint64 `long:"crash-with-snapshot-at"`
	DebugCrashFile string `long:"snapshot-dump-path"`
}

// Config represent the configuration of the processor package.
type Config struct {
	Level               encoding.LogLevel `long:"log-level"`
	LogOrderSubmitDebug encoding.Bool     `long:"log-order-submit-debug"`
	LogOrderAmendDebug  encoding.Bool     `long:"log-order-amend-debug"`
	LogOrderCancelDebug encoding.Bool     `long:"log-order-cancel-debug"`
	Ratelimit           ratelimit.Config  `group:"Ratelimit"             namespace:"ratelimit"`
	KeepCheckpointsMax  uint              `long:"keep-checkpoints-max"`
	SnapshotDebug       Snapshot          `group:"SnapshotDebug"         namespace:"snapshotdebug"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:               encoding.LogLevel{Level: logging.InfoLevel},
		LogOrderSubmitDebug: true,
		Ratelimit:           ratelimit.NewDefaultConfig(),
		KeepCheckpointsMax:  20,
		SnapshotDebug: Snapshot{
			DevEnabled:     false,
			CrashAtHeight:  0,
			DebugCrashFile: "/tmp/snapshot.json",
		},
	}
}
