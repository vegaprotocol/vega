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
	"errors"

	"code.vegaprotocol.io/vega/vegatools/events"
)

type eventsCmd struct {
	OutputFile string `description:"file to write json events to"  long:"out"    short:"o"`
	EventsFile string `description:"event file to parse into json" long:"events" short:"e"`
}

func (opts *eventsCmd) Execute(_ []string) error {
	if opts.OutputFile == "" {
		return errors.New("--out must be specified")
	}
	if opts.EventsFile == "" {
		return errors.New("--events must be specified")
	}
	return events.Run(opts.EventsFile, opts.OutputFile)
}
