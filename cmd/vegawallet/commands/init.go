package cmd

import (
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
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

type InitHandler func(home string, f *InitFlags) (*InitResponse, error)

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
			resp, err := handler(rf.Home, f)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintInitResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
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
	Force bool
}

type InitResponse struct {
	RSAKeys struct {
		PublicKeyFilePath  string `json:"publicKeyFilePath"`
		PrivateKeyFilePath string `json:"privateKeyFilePath"`
	} `json:"rsaKeys"`
}

func Init(home string, f *InitFlags) (*InitResponse, error) {
	_, err := wallets.InitialiseStore(home)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise wallets store: %w", err)
	}

	svcStore, err := svcstore.InitialiseStore(paths.New(home))
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise service store: %w", err)
	}

	if err = service.InitialiseService(svcStore, f.Force); err != nil {
		return nil, fmt.Errorf("couldn't initialise the service: %w", err)
	}

	resp := &InitResponse{}
	pubRSAKeysPath, privRSAKeysPath := svcStore.GetRSAKeysPath()
	resp.RSAKeys.PublicKeyFilePath = pubRSAKeysPath
	resp.RSAKeys.PrivateKeyFilePath = privRSAKeysPath

	return resp, nil
}

func PrintInitResponse(w io.Writer, resp *InitResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().Text("Service public RSA keys created at: ").SuccessText(resp.RSAKeys.PublicKeyFilePath).NextLine()
	str.CheckMark().Text("Service private RSA keys created at: ").SuccessText(resp.RSAKeys.PrivateKeyFilePath).NextLine()
	str.CheckMark().SuccessText("Initialisation succeeded").NextSection()

	str.BlueArrow().InfoText("Create a wallet").NextLine()
	str.Text("To create a wallet, use the following command:").NextSection()
	str.Code(fmt.Sprintf("%s create --wallet \"YOUR_USERNAME\"", os.Args[0])).NextSection()
	str.Text("The ").Bold("--wallet").Text(" flag sets the name of your wallet and will be used to login to Vega Console.").NextSection()
	str.Text("For more information, use ").Bold("--help").Text(" flag.").NextLine()
}
