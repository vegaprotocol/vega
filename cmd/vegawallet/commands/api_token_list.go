package cmd

import (
	"errors"
	"fmt"
	"io"
	"time"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	tokenStoreV1 "code.vegaprotocol.io/vega/wallet/service/v2/connections/store/v1"
	"github.com/spf13/cobra"
)

var (
	listAPITokensLong = cli.LongDesc(`
		List all the registered long-living API tokens
	`)

	listAPITokensExample = cli.Examples(`
		# List the long-living API tokens
		{{.Software}} api-token list
	`)
)

type ListAPITokensHandler func(f ListAPITokensFlags) (connections.ListAPITokensResult, error)

func NewCmdListAPITokens(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(f ListAPITokensFlags) (connections.ListAPITokensResult, error) {
		vegaPaths := paths.New(rf.Home)

		tokenStore, err := tokenStoreV1.InitialiseStore(vegaPaths, f.passphrase)
		if err != nil {
			if errors.Is(err, api.ErrWrongPassphrase) {
				return connections.ListAPITokensResult{}, err
			}
			return connections.ListAPITokensResult{}, fmt.Errorf("couldn't load the token store: %w", err)
		}
		defer tokenStore.Close()

		return connections.ListAPITokens(tokenStore)
	}

	return BuildCmdListAPITokens(w, ensureAPITokenStoreIsInit, h, rf)
}

func BuildCmdListAPITokens(w io.Writer, preCheck APITokePreCheck, handler ListAPITokensHandler, rf *RootFlags) *cobra.Command {
	f := &ListAPITokensFlags{}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all the registered long-living API tokens",
		Long:    listAPITokensLong,
		Example: listAPITokensExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := preCheck(rf); err != nil {
				return err
			}

			if err := f.Validate(); err != nil {
				return err
			}

			res, err := handler(*f)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				printListAPITokens(w, res)
			case flags.JSONOutput:
				return printer.FprintJSON(w, res)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&f.PassphraseFile,
		"passphrase-file",
		"",
		"Path to the file containing the tokens database passphrase",
	)

	return cmd
}

type ListAPITokensFlags struct {
	PassphraseFile string
	passphrase     string
}

func (f *ListAPITokensFlags) Validate() error {
	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return err
	}
	f.passphrase = passphrase
	return nil
}

func printListAPITokens(w io.Writer, resp connections.ListAPITokensResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	if len(resp.Tokens) == 0 {
		str.InfoText("No tokens registered.").NextLine()
		return
	}

	for i, token := range resp.Tokens {
		str.Text("- ").WarningText(token.Token.String()).NextLine()
		if token.Description != "" {
			str.Text("  ").Text(token.Description).NextLine()
		}
		str.Pad().Text("Created at: ").Text(token.CreationDate.String())
		if token.ExpirationDate != nil {
			str.NextLine().Pad().Text("Expiration date: ").Text(token.ExpirationDate.String())
			if !token.ExpirationDate.After(time.Now()) {
				str.Text(" (expired)")
			}
		}

		if i == len(resp.Tokens)-1 {
			str.NextLine()
		} else {
			str.NextSection()
		}
	}
}
