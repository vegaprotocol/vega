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

package tools

import (
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/vegatools/stream"
)

type streamCmd struct {
	config.OutputFlag
	BatchSize uint     `description:"size of the event stream batch of events" long:"batch-size"                                                                               short:"b"`
	Party     string   `description:"name of the party to listen for updates"  long:"party"                                                                                    short:"p"`
	Market    string   `description:"name of the market to listen for updates" long:"market"                                                                                   short:"m"`
	Address   string   `description:"address of the grpc server"               long:"address"                                                                                  required:"true"   short:"a"`
	LogFormat string   `default:"raw"                                          description:"output stream data in specified format. Allowed values: raw (default), text, json" long:"log-format"`
	Reconnect bool     `description:"if connection dies, attempt to reconnect" long:"reconnect"                                                                                short:"r"`
	Type      []string `default:""                                             description:"one or more event types to subscribe to (default=ALL)"                             long:"type"       short:"t"`
}

func (opts *streamCmd) Execute(_ []string) error {
	return stream.Run(
		opts.BatchSize,
		opts.Party,
		opts.Market,
		opts.Address,
		opts.LogFormat,
		opts.Reconnect,
		opts.Type,
	)
}
