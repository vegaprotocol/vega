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

	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cmtdebug "github.com/cometbft/cometbft/cmd/cometbft/commands/debug"
	cmtcfg "github.com/cometbft/cometbft/config"
	cmtcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/jessevdk/go-flags"
)

func Tm(_ context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"tm",
		"deprecated, see vega cometbft instead",
		"deprecated, see vega cometbft instead",
		&cometbftCmd{},
	)

	return err
}

func Tendermint(_ context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"tendermint",
		"deprecated, see vega cometbft instead",
		"deprecated, see vega cometbft instead",
		&cometbftCmd{},
	)

	return err
}

func CometBFT(_ context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"cometbft",
		"Run cometbft commands",
		"Run cometbft commands",
		&cometbftCmd{},
	)

	return err
}

type cometbftCmd struct{}

func (opts *cometbftCmd) Execute(_ []string) error {
	os.Args = os.Args[1:]
	rootCmd := cmtcmd.RootCmd
	rootCmd.AddCommand(
		cmtcmd.GenValidatorCmd,
		cmtcmd.InitFilesCmd,
		cmtcmd.LightCmd,
		// unsupported
		// cmtcmd.ReplayCmd,
		// cmtcmd.ReplayConsoleCmd,
		cmtcmd.ResetAllCmd,
		cmtcmd.ResetPrivValidatorCmd,
		cmtcmd.ResetStateCmd,
		cmtcmd.ShowValidatorCmd,
		cmtcmd.TestnetFilesCmd,
		cmtcmd.ShowNodeIDCmd,
		cmtcmd.GenNodeKeyCmd,
		cmtcmd.VersionCmd,
		cmtcmd.RollbackStateCmd,
		cmtcmd.CompactGoLevelDBCmd,
		cmtdebug.DebugCmd,
		cmtcli.NewCompletionCmd(rootCmd, true),
	)

	baseCmd := cmtcli.PrepareBaseCmd(rootCmd, "CMT", os.ExpandEnv(filepath.Join("$HOME", cmtcfg.DefaultCometDir)))
	if err := baseCmd.Execute(); err != nil {
		return err
	}

	return nil
}
