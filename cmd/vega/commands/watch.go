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

package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/jessevdk/go-flags"
)

type watch struct {
	Address    string `default:"tcp://0.0.0.0:26657" description:"Node address" long:"address" short:"a"`
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
