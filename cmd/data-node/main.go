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

	"code.vegaprotocol.io/data-node/config"
	"github.com/jessevdk/go-flags"
)

var (
	// VersionHash specifies the git commit used to build the application. See VERSION_HASH in Makefile for details.
	CLIVersionHash = ""

	// Version specifies the version used to build the application. See VERSION in Makefile for details.
	CLIVersion = "v0.53.0+dev"
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
	parser := flags.NewParser(&config.Empty{}, flags.Default)

	if err := Register(ctx, parser,
		Init,
		Gateway,
		Node,
		Version,
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
