// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package logging

// Config contains the configurable items for this package.
type Config struct {
	Environment string  `long:"env" choice:"dev" choice:"custom"`
	Custom      *Custom `tomlcp:"This section takes effect only when Environment is set to \"custom\"."`
}

// Custom contains the custom log config.
type Custom struct {
	Zap        *Zap
	ZapEncoder *ZapEncoder
}

// Zap configures a ZapConfig.
type Zap struct {
	Level            Level
	Development      bool
	Encoding         string // console or json
	OutputPaths      []string
	ErrorOutputPaths []string
}

// ZapEncoder configures a ZapEncoderConfig.
type ZapEncoder struct {
	CallerKey      string
	EncodeCaller   string
	EncodeDuration string
	EncodeLevel    string
	EncodeName     string
	EncodeTime     string
	LevelKey       string
	LineEnding     string
	MessageKey     string
	NameKey        string
	TimeKey        string
}

// NewDefaultConfig creates an instance of the package-specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Environment: "dev",
		Custom: &Custom{
			Zap: &Zap{
				Development:      true,
				Encoding:         "console",
				Level:            DebugLevel,
				OutputPaths:      []string{"stdout"},
				ErrorOutputPaths: []string{"stderr"},
			},
			ZapEncoder: &ZapEncoder{
				CallerKey:      "C",
				EncodeCaller:   "short",
				EncodeDuration: "string",
				EncodeLevel:    "capital",
				EncodeName:     "full",
				EncodeTime:     "iso8601",
				LevelKey:       "L",
				LineEnding:     "\n",
				MessageKey:     "M",
				NameKey:        "N",
				TimeKey:        "T",
			},
		},
	}
}
