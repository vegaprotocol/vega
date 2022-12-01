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
	vgterm "code.vegaprotocol.io/vega/libs/term"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	tokenStore "code.vegaprotocol.io/vega/wallet/api/session/store/v1"
	"github.com/spf13/cobra"
)

var (
	deleteAPITokenLong = cli.LongDesc(`
		Delete a long-living API token
	`)

	deleteAPITokenExample = cli.Examples(`
		# Delete a long-living API token
		{{.Software}} api-token delete --token TOKEN

		# Delete a long-living API token without asking for confirmation
		{{.Software}} api-token delete --token TOKEN --force
	`)
)

type DeleteAPITokenHandler func(f DeleteAPITokenFlags, params api.AdminDeleteAPITokenParams) error

func NewCmdDeleteAPIToken(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(f DeleteAPITokenFlags, params api.AdminDeleteAPITokenParams) error {
		vegaPaths := paths.New(rf.Home)

		store, err := tokenStore.LoadStore(vegaPaths, f.passphrase)
		if err != nil {
			if errors.Is(err, api.ErrWrongPassphrase) {
				return err
			}
			return fmt.Errorf("couldn't load the tokens store: %w", err)
		}

		deleteAPIToken := api.NewAdminDeleteAPIToken(store)
		if _, errorDetails := deleteAPIToken.Handle(context.Background(), params, jsonrpc.RequestMetadata{}); errorDetails != nil {
			return errors.New(errorDetails.Data)
		}
		return nil
	}

	return BuildCmdDeleteAPIToken(w, ensureAPITokensStoreIsInit, h, rf)
}

func BuildCmdDeleteAPIToken(w io.Writer, preCheck APITokePreCheck, handler DeleteAPITokenHandler, rf *RootFlags) *cobra.Command {
	f := &DeleteAPITokenFlags{}

	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete a long-living API token",
		Long:    deleteAPITokenLong,
		Example: deleteAPITokenExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := preCheck(rf); err != nil {
				return err
			}

			params, err := f.Validate()
			if err != nil {
				return err
			}

			if !f.Force && vgterm.HasTTY() {
				if !flags.AreYouSure() {
					return nil
				}
			}

			if err := handler(*f, params); err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				printDeletedAPIToken(w)
			case flags.JSONOutput:
				return nil
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&f.Token,
		"token",
		"",
		"Token to delete",
	)
	cmd.Flags().StringVar(&f.PassphraseFile,
		"passphrase-file",
		"",
		"Path to the file containing the tokens database passphrase",
	)
	cmd.Flags().BoolVarP(&f.Force,
		"force", "f",
		false,
		"Do not ask for confirmation",
	)

	return cmd
}

type DeleteAPITokenFlags struct {
	Token          string
	PassphraseFile string
	Force          bool
	passphrase     string
}

func (f *DeleteAPITokenFlags) Validate() (api.AdminDeleteAPITokenParams, error) {
	if len(f.Token) == 0 {
		return api.AdminDeleteAPITokenParams{}, flags.MustBeSpecifiedError("token")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminDeleteAPITokenParams{}, err
	}
	f.passphrase = passphrase

	if !f.Force && vgterm.HasNoTTY() {
		return api.AdminDeleteAPITokenParams{}, ErrForceFlagIsRequiredWithoutTTY
	}

	return api.AdminDeleteAPITokenParams{
		Token: f.Token,
	}, nil
}

func printDeletedAPIToken(w io.Writer) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("The API token has been successfully deleted.").NextLine()
}
