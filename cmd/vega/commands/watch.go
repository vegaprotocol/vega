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

package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
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
		return errors.New("error: watch requires at least one filter")
	}

	c, err := abci.NewClient(opts.Address)
	if err != nil {
		return fmt.Errorf("could not instantiate abci client: %w", err)
	}

	ctx := context.Background()
	fn := func(e tmctypes.ResultEvent) error {
		bz, err := json.Marshal(e.Data)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", bz)
		return nil
	}
	if err := c.Subscribe(ctx, fn, args...); err != nil {
		return err
	}

	return nil
}

func Watch(ctx context.Context, parser *flags.Parser) error {
	var (
		shortDesc = "Watches events from Tendermint"
		longDesc  = `Events results are encoded in JSON and can be filtered
using a simple query language.  You can use one or more filters.
See https://docs.tendermint.com/master/app-dev/subscribing-to-events-via-websocket.html
for more information about the query syntax.

Example:
watch "tm.event = 'NewBlock'" "tm.event = 'Transaction'"`
	)
	_, err := parser.AddCommand("watch", shortDesc, longDesc, &watch{})
	return err
}
