package cmd

import (
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	vgterm "code.vegaprotocol.io/vega/libs/term"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	tokenStoreV1 "code.vegaprotocol.io/vega/wallet/service/v2/connections/store/longliving/v1"
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

type DeleteAPITokenHandler func(f DeleteAPITokenFlags) error

func NewCmdDeleteAPIToken(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(f DeleteAPITokenFlags) error {
		vegaPaths := paths.New(rf.Home)

		tokenStore, err := tokenStoreV1.InitialiseStore(vegaPaths, f.passphrase)
		if err != nil {
			if errors.Is(err, api.ErrWrongPassphrase) {
				return err
			}
			return fmt.Errorf("couldn't load the token store: %w", err)
		}
		defer tokenStore.Close()

		return connections.DeleteAPIToken(tokenStore, f.Token)
	}

	return BuildCmdDeleteAPIToken(w, ensureAPITokenStoreIsInit, h, rf)
}

func BuildCmdDeleteAPIToken(w io.Writer, preCheck APITokenPreCheck, handler DeleteAPITokenHandler, rf *RootFlags) *cobra.Command {
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

			if err := f.Validate(); err != nil {
				return err
			}

			if !f.Force && vgterm.HasTTY() {
				if !flags.AreYouSure() {
					return nil
				}
			}

			if err := handler(*f); err != nil {
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

func (f *DeleteAPITokenFlags) Validate() error {
	if len(f.Token) == 0 {
		return flags.MustBeSpecifiedError("token")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return err
	}
	f.passphrase = passphrase

	if !f.Force && vgterm.HasNoTTY() {
		return ErrForceFlagIsRequiredWithoutTTY
	}

	return nil
}

func printDeletedAPIToken(w io.Writer) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("The API token has been successfully deleted.").NextLine()
}
