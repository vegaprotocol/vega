package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	tokenStore "code.vegaprotocol.io/vega/wallet/api/session/store/v1"
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

type ListAPITokensHandler func(f ListAPITokensFlags) (api.AdminListAPITokensResult, error)

func NewCmdListAPITokens(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(f ListAPITokensFlags) (api.AdminListAPITokensResult, error) {
		vegaPaths := paths.New(rf.Home)

		store, err := tokenStore.LoadStore(vegaPaths, f.passphrase)
		if err != nil {
			if errors.Is(err, api.ErrWrongPassphrase) {
				return api.AdminListAPITokensResult{}, err
			}
			return api.AdminListAPITokensResult{}, fmt.Errorf("couldn't load the tokens store: %w", err)
		}

		listAPITokens := api.NewAdminListAPITokens(store)
		rawResult, errorDetails := listAPITokens.Handle(context.Background(), nil, jsonrpc.RequestMetadata{})
		if errorDetails != nil {
			return api.AdminListAPITokensResult{}, errors.New(errorDetails.Data)
		}
		return rawResult.(api.AdminListAPITokensResult), nil
	}

	return BuildCmdListAPITokens(w, ensureAPITokensStoreIsInit, h, rf)
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

func printListAPITokens(w io.Writer, resp api.AdminListAPITokensResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	if len(resp.Tokens) == 0 {
		str.InfoText("No tokens registered.").NextLine()
		return
	}

	for i, token := range resp.Tokens {
		str.Text("- ").WarningText(token.Token).NextLine()
		if token.Description != "" {
			str.Text("  ").Text(token.Description).NextLine()
		}
		str.Pad().Text("Created at: ").Text(token.CreateAt.String())

		if i == len(resp.Tokens)-1 {
			str.NextLine()
		} else {
			str.NextSection()
		}
	}
}
