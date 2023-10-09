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

package grpc

import (
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

var namedLogger = "api.grpc"

type Config struct {
	Reflection         encoding.Bool     `description:"Enable GRPC reflection, required for grpc-ui"                              long:"reflection"`
	Level              encoding.LogLevel `choice:"debug"                                                                          choice:"info"            choice:"warning" long:"log-level"`
	MaxPageSizeDefault uint32            `description:"How many results to return per page if client does not specify explicitly" long:"default-page-size"`
}

func NewDefaultConfig() Config {
	return Config{
		Reflection:         encoding.Bool(true),
		Level:              encoding.LogLevel{Level: logging.InfoLevel},
		MaxPageSizeDefault: 50,
	}
}
