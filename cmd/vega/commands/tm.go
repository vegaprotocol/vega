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
