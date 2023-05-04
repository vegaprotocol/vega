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
	tokenStoreV1 "code.vegaprotocol.io/vega/wallet/service/v2/connections/store/longliving/v1"
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

type DescribeAPITokenHandler func(f DescribeAPITokenFlags) (connections.TokenDescription, error)

func NewCmdDescribeAPIToken(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(f DescribeAPITokenFlags) (connections.TokenDescription, error) {
		vegaPaths := paths.New(rf.Home)

		tokenStore, err := tokenStoreV1.InitialiseStore(vegaPaths, f.passphrase)
		if err != nil {
			if errors.Is(err, api.ErrWrongPassphrase) {
				return connections.TokenDescription{}, err
			}
			return connections.TokenDescription{}, fmt.Errorf("couldn't load the token store: %w", err)
		}
		defer tokenStore.Close()

		return connections.DescribeAPIToken(tokenStore, f.Token)
	}

	return BuildCmdDescribeAPIToken(w, ensureAPITokenStoreIsInit, h, rf)
}

func BuildCmdDescribeAPIToken(w io.Writer, preCheck APITokenPreCheck, handler DescribeAPITokenHandler, rf *RootFlags) *cobra.Command {
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

			err := f.Validate()
			if err != nil {
				return err
			}

			res, err := handler(*f)
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

func (f *DescribeAPITokenFlags) Validate() error {
	if len(f.Token) == 0 {
		return flags.MustBeSpecifiedError("token")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return err
	}
	f.passphrase = passphrase

	return nil
}

func printDescribeAPIToken(w io.Writer, resp connections.TokenDescription) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("Token:").NextLine()
	str.WarningText(resp.Token.String()).NextSection()

	if len(resp.Description) != 0 {
		str.Text("Description:").NextLine()
		str.WarningText(resp.Description).NextSection()
	}
	str.Text("Creation date:").NextLine()
	str.WarningText(resp.CreationDate.String()).NextSection()

	if resp.ExpirationDate != nil {
		str.Text("Expiration date:").NextLine()
		str.WarningText(resp.ExpirationDate.String())
		if !resp.ExpirationDate.After(time.Now()) {
			str.DangerBold(" (expired)")
		}
		str.NextSection()
	}

	str.Text("This token is linked to the wallet ").WarningText(resp.Wallet.Name).Text(".").NextLine()
}
