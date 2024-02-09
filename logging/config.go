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

package logging

// Config contains the configurable items for this package.
type Config struct {
	Environment string  `choice:"dev"                                                                   choice:"custom" long:"env"`
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
