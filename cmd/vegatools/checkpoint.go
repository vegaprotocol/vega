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
	"code.vegaprotocol.io/vega/vegatools/checkpoint"
)

type checkpointCmd struct {
	config.OutputFlag

	InPath   string `description:"input file to parse"                                              long:"file"     required:"true" short:"f"`
	OutPath  string `description:"output file to write to [default is STDOUT]"                      long:"out"      short:"o"`
	Validate bool   `description:"validate contents of the checkpoint file"                         long:"validate" short:"v"`
	Generate bool   `description:"The chain to be imported"                                         long:"generate" short:"g"`
	Dummy    bool   `description:"generate a dummy file [added for debugging, but could be useful]" long:"dummy"    short:"d"`
}

func (opts *checkpointCmd) Execute(_ []string) error {
	checkpoint.Run(
		opts.InPath,
		opts.OutPath,
		opts.Generate,
		opts.Validate,
		opts.Dummy,
	)
	return nil
}
