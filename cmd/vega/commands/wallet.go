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

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
	"github.com/jessevdk/go-flags"
)

type walletCmd struct{}

func (opts *walletCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]

	writer := &cmd.Writer{
		Out: os.Stdout,
		Err: os.Stderr,
	}
	cmd.Execute(writer)

	return nil
}

func Wallet(ctx context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"wallet",
		"Run vega wallet",
		"Run the vega wallet",
		&walletCmd{},
	)

	return err
}
