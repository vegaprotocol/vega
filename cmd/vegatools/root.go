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
	"context"

	"code.vegaprotocol.io/vega/core/config"

	"github.com/jessevdk/go-flags"
)

type RootCmd struct {
	// Global options
	config.VegaHomeFlag

	// Subcommands
	Snapshot   snapshotCmd   `command:"snapshot"   description:"Display information about saved snapshots"`
	Checkpoint checkpointCmd `command:"checkpoint" description:"Make checkpoint human-readable, or generate checkpoint from human readable format"`
	Stream     streamCmd     `command:"stream"     description:"Stream events from vega node"`
	CheckTx    checkTxCmd    `command:"check-tx"   description:"Check an encoded transaction from a dependent app is unmarshalled and re-encoded back to the same encoded value. Checks data integrity"`
	Events     eventsCmd     `command:"events"     description:"Parse event files written by a core node and write them in humna-readble form"`
}

var rootCmd RootCmd

func VegaTools(ctx context.Context, parser *flags.Parser) error {
	rootCmd = RootCmd{
		Snapshot:   snapshotCmd{},
		Checkpoint: checkpointCmd{},
		Stream:     streamCmd{},
		CheckTx:    checkTxCmd{},
		Events:     eventsCmd{},
	}

	var (
		short = "useful tooling for probing a vega node and its state"
		long  = `useful tooling for probing a vega node and its state`
	)
	_, err := parser.AddCommand("tools", short, long, &rootCmd)
	return err
}
