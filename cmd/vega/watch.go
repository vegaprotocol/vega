package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	tmctypes "github.com/tendermint/tendermint/rpc/core/types"

	"code.vegaprotocol.io/vega/blockchain/tm"
)

type watchCommand struct {
	command

	addr string
}

func (w *watchCommand) Init(c *Cli) {
	w.cli = c
	w.cmd = &cobra.Command{
		Use:     "watch",
		Short:   "Watches events from the tendermint node",
		Example: `watch "tm.event = 'NewBlock'"`,
		Long: `Events are encoded in JSON and can be filtered using a simple query language.
See: https://docs.tendermint.com/master/app-dev/subscribing-to-events-via-websocket.html for more information about the query syntax.`,
		RunE: w.run,
		Args: cobra.ExactArgs(1),
	}

	addr := tm.NewDefaultConfig().ClientAddr
	w.cmd.Flags().StringVarP(&w.addr, "addr", "a", addr, "Node Address")
}

func (w *watchCommand) run(cmd *cobra.Command, args []string) error {
	cfg := tm.NewDefaultConfig()
	cfg.ClientAddr = w.addr

	c, err := tm.NewClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	fn := func(e tmctypes.ResultEvent) error {
		bz, err := json.Marshal(e.Data)
		if err != nil {
			return err
		}
		fmt.Printf("%s", bz)
		return nil
	}
	if err := c.Subscribe(ctx, args[0], fn); err != nil {
		return err
	}

	return nil
}
