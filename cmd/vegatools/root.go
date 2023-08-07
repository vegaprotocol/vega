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
	CheckTx    checkTxCmd    `command:"check-tx"    description:"Decode transactions created from a dependent app, check vega's decoded transaction matches your apps transaction"`
}

var rootCmd RootCmd

func VegaTools(ctx context.Context, parser *flags.Parser) error {
	rootCmd = RootCmd{
		Snapshot:   snapshotCmd{},
		Checkpoint: checkpointCmd{},
		Stream:     streamCmd{},
		CheckTx:    checkTxCmd{},
	}

	var (
		short = "useful tooling for probing a vega node and its state"
		long  = `useful tooling for probing a vega node and its state`
	)
	_, err := parser.AddCommand("tools", short, long, &rootCmd)
	return err
}
