package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/blockchain/abci"
	"github.com/jessevdk/go-flags"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type watch struct {
	Address    string `short:"a" long:"address" description:"Node address" default:"tcp://0.0.0.0:26657"`
	Positional struct {
		Filters []string `positional-arg-name:"<FILTERS>"`
	} `positional-args:"true"`
}

func (opts *watch) Execute(_ []string) error {
	args := opts.Positional.Filters
	if len(args) == 0 {
		return errors.New("Error: watch requires at least one filter")
	}

	c, err := abci.NewClient(opts.Address)
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
	if err := c.Subscribe(ctx, fn, args...); err != nil {
		return err
	}

	return nil
}

func Watch(parser *flags.Parser) error {
	var (
		shortDesc = "Watches events from Tendermint"
		longDesc  = `Events results are encoded in JSON and can be filtered
using a simple query language.  You can use one or more filters.
See https://docs.tendermint.com/master/app-dev/subscribing-to-events-via-websocket.html
for more information about the query syntax.

Example:
watch "tm.event = 'NewBlock'" "tm.event = 'Tx'"`
	)
	_, err := parser.AddCommand("watch", shortDesc, longDesc, &watch{})
	return err
}
