package cmd

import (
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/paths"
	tokenStoreV1 "code.vegaprotocol.io/vega/wallet/service/v2/connections/store/longliving/v1"
	"github.com/spf13/cobra"
)

var (
	apiTokenInitLong = cli.LongDesc(`
		Initialise the system supporting long-living API tokens
	`)

	apiTokenInitExample = cli.Examples(`
		# Initialise the system supporting long-living API tokens
		{{.Software}} api-token init
	`)

	tokenPassphraseOptions = flags.PassphraseOptions{
		Name:        "tokens store",
		Description: "This passphrase is used to encrypt the long-living connection tokens.\nThis passphrase will be asked to start the wallet service.",
	}
)

type APITokenInitHandler func(home string, f *InitAPITokenFlags) (bool, error)

func NewCmdInitAPIToken(w io.Writer, rf *RootFlags) *cobra.Command {
	return BuildCmdInitAPIToken(w, InitAPIToken, rf)
}

func BuildCmdInitAPIToken(w io.Writer, handler APITokenInitHandler, rf *RootFlags) *cobra.Command {
	f := &InitAPITokenFlags{}

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialise the system supporting long-living API tokens",
		Long:    apiTokenInitLong,
		Example: apiTokenInitExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			initialized, err := handler(rf.Home, f)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintAPITokenInitResponse(w, initialized)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&f.PassphraseFile,
		"passphrase-file",
		"",
		"Path to the file containing the tokens database passphrase",
	)

	cmd.Flags().BoolVarP(&f.Force,
		"force", "f",
		false,
		"Remove the existing token database and recreate it",
	)

	return cmd
}

type InitAPITokenFlags struct {
	PassphraseFile string
	Force          bool
}

func InitAPIToken(home string, f *InitAPITokenFlags) (bool, error) {
	vegaPaths := paths.New(home)

	// Verify the init state of the tokens store
	init, err := tokenStoreV1.IsStoreBootstrapped(vegaPaths)
	if err != nil {
		return false, fmt.Errorf("could not verify the initialization state of the tokens store: %w", err)
	}
	if init && !f.Force {
		return false, nil
	}

	passphrase, err := flags.GetConfirmedPassphraseWithContext(tokenPassphraseOptions, f.PassphraseFile)
	if err != nil {
		return false, err
	}
	tokenStore, err := tokenStoreV1.ReinitialiseStore(vegaPaths, passphrase)
	if err != nil {
		return false, fmt.Errorf("couldn't initialise the tokens store: %w", err)
	}
	tokenStore.Close()
	return true, nil
}

func PrintAPITokenInitResponse(w io.Writer, init bool) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	if init {
		str.CheckMark().SuccessText("Support for long-living tokens has been initialised.").NextSection()
	} else {
		str.CheckMark().SuccessText("Support for long-living tokens has ").SuccessBold("already").SuccessText(" been initialised.").NextSection()
	}

	str.BlueArrow().InfoText("Generate a long-living API token").NextLine()
	str.Text("To generate a long-living API token, use the following command:").NextSection()
	str.Code(fmt.Sprintf("%s api-token generate --wallet-name \"WALLET_ASSOCIATED_TO_THE_TOKEN\"", os.Args[0])).NextSection()
	str.Text("For more information, use ").Bold("--help").Text(" flag.").NextLine()
}
