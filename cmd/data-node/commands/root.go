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
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/datanode/config"

	"github.com/jessevdk/go-flags"
)

// Subcommand is the signature of a sub command that can be registered.
type Subcommand func(context.Context, *flags.Parser) error

// Register registers one or more subcommands.
func Register(ctx context.Context, parser *flags.Parser, cmds ...Subcommand) error {
	for _, fn := range cmds {
		if err := fn(ctx, parser); err != nil {
			return err
		}
	}
	return nil
}

func Execute(ctx context.Context) error {
	parser := flags.NewParser(&config.Empty{}, flags.Default)

	if err := Register(ctx, parser,
		Init,
		Gateway,
		Node,
		Start,
		Version,
		Postgres,
		ListSnapshots,
	); err != nil {
		fmt.Printf("%+v\n", err)
		return err
	}

	if _, err := parser.Parse(); err != nil {
		switch t := err.(type) {
		case *flags.Error:
			if t.Type != flags.ErrHelp {
				parser.WriteHelp(os.Stdout)
			}
		}
		return err
	}
	return nil
}
