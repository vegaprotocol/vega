package cmd

import (
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/service"
	svcstore "code.vegaprotocol.io/vega/wallet/service/store/v1"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	initLong = cli.LongDesc(`
		Creates the folders, the configuration files and RSA keys needed by the service
		to operate.
	`)

	initExample = cli.Examples(`
		# Initialise the software
		{{.Software}} init

		# Re-initialise the software
		{{.Software}} init --force
	`)
)

type InitHandler func(home string, f *InitFlags) error

func NewCmdInit(w io.Writer, rf *RootFlags) *cobra.Command {
	return BuildCmdInit(w, Init, rf)
}

func BuildCmdInit(w io.Writer, handler InitHandler, rf *RootFlags) *cobra.Command {
	f := &InitFlags{}

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialise the software",
		Long:    initLong,
		Example: initExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := handler(rf.Home, f); err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintInitResponse(w)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&f.Force,
		"force", "f",
		false,
		"Overwrite exiting wallet configuration at the specified path",
	)

	return cmd
}

type InitFlags struct {
	Force                bool
	TokensPassphraseFile string
}

func Init(home string, f *InitFlags) error {
	walletStore, err := wallets.InitialiseStore(home, false)
	if err != nil {
		return fmt.Errorf("couldn't initialise wallets store: %w", err)
	}
	defer walletStore.Close()

	vegaPaths := paths.New(home)

	svcStore, err := svcstore.InitialiseStore(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise service store: %w", err)
	}

	if err = service.InitialiseService(svcStore, f.Force); err != nil {
		return fmt.Errorf("couldn't initialise the service: %w", err)
	}

	return nil
}

func PrintInitResponse(w io.Writer) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("Initialisation succeeded").NextSection()

	str.BlueArrow().InfoText("Create a wallet").NextLine()
	str.Text("To create a wallet, use the following command:").NextSection()
	str.Code(fmt.Sprintf("%s create --wallet \"YOUR_WALLET\"", os.Args[0])).NextSection()
	str.Text("For more information, use ").Bold("--help").Text(" flag.").NextLine()
}
