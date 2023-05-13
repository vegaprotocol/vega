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
	"os"

	cmd "code.vegaprotocol.io/vega/cmd/data-node/commands"
	"github.com/jessevdk/go-flags"
)

type datanodeCmd struct{}

func (opts *datanodeCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]
	return cmd.Execute(context.Background())
}

func Datanode(ctx context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"datanode",
		"The vega data node",
		"The vega data node",
		&datanodeCmd{},
	)

	return err
}
