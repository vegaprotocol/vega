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
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	tmcmd "github.com/tendermint/tendermint/cmd/cometbft/commands"
	tmdebug "github.com/tendermint/tendermint/cmd/cometbft/commands/debug"
	tmcfg "github.com/tendermint/tendermint/config"
	tmcli "github.com/tendermint/tendermint/libs/cli"
)

func Tm(_ context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"tm",
		"deprecated, see vega tendermint instead",
		"deprecated, see vega tendermint instead",
		&tmCmd{},
	)

	return err
}

func Tendermint(_ context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"tendermint",
		"Run tendermint commands",
		"Run tendermint commands",
		&tmCmd{},
	)

	return err
}

type tmCmd struct{}

func (opts *tmCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]
	rootCmd := tmcmd.RootCmd
	rootCmd.AddCommand(
		tmcmd.GenValidatorCmd,
		tmcmd.InitFilesCmd,
		tmcmd.ProbeUpnpCmd,
		tmcmd.LightCmd,
		tmcmd.ReplayCmd,
		tmcmd.ReplayConsoleCmd,
		tmcmd.ResetAllCmd,
		tmcmd.ResetPrivValidatorCmd,
		tmcmd.ResetStateCmd,
		tmcmd.ShowValidatorCmd,
		tmcmd.TestnetFilesCmd,
		tmcmd.ShowNodeIDCmd,
		tmcmd.GenNodeKeyCmd,
		tmcmd.VersionCmd,
		tmcmd.RollbackStateCmd,
		tmcmd.CompactGoLevelDBCmd,
		tmdebug.DebugCmd,
		tmcli.NewCompletionCmd(rootCmd, true),
	)

	baseCmd := tmcli.PrepareBaseCmd(rootCmd, "TM", os.ExpandEnv(filepath.Join("$HOME", tmcfg.DefaultTendermintDir)))
	if err := baseCmd.Execute(); err != nil {
		return err
	}

	return nil
}
