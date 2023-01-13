package cmd

import (
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/paths"
	tokenStoreV1 "code.vegaprotocol.io/vega/wallet/service/v2/connections/store/v1"
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

type APITokePreCheck func(rf *RootFlags) error

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

	isInit, err := tokenStoreV1.IsStoreInitialized(vegaPaths)
	if err != nil {
		return fmt.Errorf("could not verify the initialization state of the token store: %w", err)
	}

	if !isInit {
		return ErrTokenStoreNotInitialized
	}

	return nil
}
