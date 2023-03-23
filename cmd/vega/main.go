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

package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/jessevdk/go-flags"

	"code.vegaprotocol.io/vega/cmd/vega/faucet"
	"code.vegaprotocol.io/vega/cmd/vega/genesis"
	"code.vegaprotocol.io/vega/cmd/vega/nodewallet"
	"code.vegaprotocol.io/vega/cmd/vega/paths"
	"code.vegaprotocol.io/vega/config"
)

var (
	CLIVersionHash = ""
	CLIVersion     = "v0.53.2"
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

func main() {
	setCommitHash()
	ctx := context.Background()
	if err := Main(ctx); err != nil {
		os.Exit(-1)
	}
}

func Main(ctx context.Context) error {
	// special case for the tendermint subcommand, so we bypass the command line
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "tm":
			return (&tmCmd{}).Execute(nil)
		case "wallet":
			return (&walletCmd{}).Execute(nil)
		}
	}

	parser := flags.NewParser(&config.Empty{}, flags.Default)

	if err := Register(ctx, parser,
		faucet.Faucet,
		genesis.Genesis,
		Init,
		nodewallet.NodeWallet,
		Verify,
		Version,
		Wallet,
		Watch,
		Tm,
		Query,
		Bridge,
		paths.Paths,
		UnsafeResetAll,
		SnapshotList,
		AnnounceNode,
		Start,
	); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return err
	}

	if _, err := parser.Parse(); err != nil {
		return err
	}
	return nil
}

func setCommitHash() {
	info, _ := debug.ReadBuildInfo()
	modified := false

	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			CLIVersionHash = v.Value
		}
		if v.Key == "vcs.modified" {
			modified = true
		}
	}
	if modified {
		CLIVersionHash += "-modified"
	}
}
