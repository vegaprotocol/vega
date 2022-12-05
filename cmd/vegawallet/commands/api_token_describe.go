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
	describeAPITokenLong = cli.LongDesc(`
		Describe a long-living API tokens and its configuration
	`)

	describeAPITokenExample = cli.Examples(`
		# Describe a long-living API tokens
		{{.Software}} api-token describe --token TOKEN
	`)
)

type DescribeAPITokenHandler func(f DescribeAPITokenFlags, params api.AdminDescribeAPITokenParams) (api.AdminDescribeAPITokenResult, error)

func NewCmdDescribeAPIToken(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(f DescribeAPITokenFlags, params api.AdminDescribeAPITokenParams) (api.AdminDescribeAPITokenResult, error) {
		vegaPaths := paths.New(rf.Home)

		store, err := tokenStore.LoadStore(vegaPaths, f.passphrase)
		if err != nil {
			if errors.Is(err, api.ErrWrongPassphrase) {
				return api.AdminDescribeAPITokenResult{}, err
			}
			return api.AdminDescribeAPITokenResult{}, fmt.Errorf("couldn't load the tokens store: %w", err)
		}

		describeAPIToken := api.NewAdminDescribeAPIToken(store)
		rawResult, errorDetails := describeAPIToken.Handle(context.Background(), params, jsonrpc.RequestMetadata{})
		if errorDetails != nil {
			return api.AdminDescribeAPITokenResult{}, errors.New(errorDetails.Data)
		}
		return rawResult.(api.AdminDescribeAPITokenResult), nil
	}

	return BuildCmdDescribeAPIToken(w, ensureAPITokensStoreIsInit, h, rf)
}

func BuildCmdDescribeAPIToken(w io.Writer, preCheck APITokePreCheck, handler DescribeAPITokenHandler, rf *RootFlags) *cobra.Command {
	f := &DescribeAPITokenFlags{}

	cmd := &cobra.Command{
		Use:     "describe",
		Short:   "Describe the token and its configuration",
		Long:    describeAPITokenLong,
		Example: describeAPITokenExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := preCheck(rf); err != nil {
				return err
			}

			params, err := f.Validate()
			if err != nil {
				return err
			}

			res, err := handler(*f, params)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				printDescribeAPIToken(w, res)
			case flags.JSONOutput:
				return printer.FprintJSON(w, res)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&f.Token,
		"token",
		"",
		"Token to describe",
	)
	cmd.Flags().StringVar(&f.PassphraseFile,
		"passphrase-file",
		"",
		"Path to the file containing the tokens database passphrase",
	)

	return cmd
}

type DescribeAPITokenFlags struct {
	PassphraseFile string
	Token          string
	passphrase     string
}

func (f *DescribeAPITokenFlags) Validate() (api.AdminDescribeAPITokenParams, error) {
	if len(f.Token) == 0 {
		return api.AdminDescribeAPITokenParams{}, flags.MustBeSpecifiedError("token")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminDescribeAPITokenParams{}, err
	}
	f.passphrase = passphrase

	return api.AdminDescribeAPITokenParams{
		Token: f.Token,
	}, nil
}

func printDescribeAPIToken(w io.Writer, resp api.AdminDescribeAPITokenResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("Token:").NextLine()
	str.WarningText(resp.Token).NextSection()
	if len(resp.Description) != 0 {
		str.Text("Description:").NextLine()
		str.WarningText(resp.Description).NextSection()
	}
	str.Text("Created at:").NextLine()
	str.WarningText(resp.CreatedAt.String()).NextSection()

	str.Text("This token is linked to the wallet ").WarningText(resp.Wallet).Text(".").NextLine()
}
