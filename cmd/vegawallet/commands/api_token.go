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

package cmd

import (
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/paths"
	tokenStoreV1 "code.vegaprotocol.io/vega/wallet/service/v2/connections/store/longliving/v1"

	"github.com/spf13/cobra"
)

var (
	ErrTokenStoreNotInitialized = errors.New("the token store is not initialized, call the `api-token init` command first")

	apiTokenLong = cli.LongDesc(`
		Manage the API tokens.

		These tokens can be used by third-party applications and the wallet service to access the wallets and send transactions, without human intervention.

		This is suitable for headless applications such as bots, and scripts.
	`)
)

type APITokenPreCheck func(rf *RootFlags) error

func NewCmdAPIToken(w io.Writer, rf *RootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api-token",
		Short: "Manage the API tokens",
		Long:  apiTokenLong,
	}

	cmd.AddCommand(NewCmdInitAPIToken(w, rf))
	cmd.AddCommand(NewCmdDeleteAPIToken(w, rf))
	cmd.AddCommand(NewCmdDescribeAPIToken(w, rf))
	cmd.AddCommand(NewCmdGenerateAPIToken(w, rf))
	cmd.AddCommand(NewCmdListAPITokens(w, rf))

	return cmd
}

func ensureAPITokenStoreIsInit(rf *RootFlags) error {
	vegaPaths := paths.New(rf.Home)

	isInit, err := tokenStoreV1.IsStoreBootstrapped(vegaPaths)
	if err != nil {
		return fmt.Errorf("could not verify the initialization state of the token store: %w", err)
	}

	if !isInit {
		return ErrTokenStoreNotInitialized
	}

	return nil
}
