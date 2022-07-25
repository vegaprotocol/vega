package cmd

import (
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	generateKeyLong = cli.LongDesc(`
		Generate a new Ed25519 key pair in the specified wallet.
	`)

	generateKeyExample = cli.Examples(`
		# Generate a key pair
		{{.Software}} key generate --wallet WALLET

		# Generate a key pair with additional metadata (name = my-wallet and type = validation)
		{{.Software}} key generate --wallet WALLET --meta "name:my-wallet,type:validation"

		# Generate a key pair with custom name
		{{.Software}} key generate --wallet WALLET --meta "name:my-wallet"
	`)
)

type GenerateKeyHandler func(*wallet.GenerateKeyRequest) (*wallet.GenerateKeyResponse, error)

func NewCmdGenerateKey(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(req *wallet.GenerateKeyRequest) (*wallet.GenerateKeyResponse, error) {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		return wallet.GenerateKey(s, req)
	}

	return BuildCmdGenerateKey(w, h, rf)
}

func BuildCmdGenerateKey(w io.Writer, handler GenerateKeyHandler, rf *RootFlags) *cobra.Command {
	f := &GenerateKeyFlags{}

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate a new key pair in a given wallet",
		Long:    generateKeyLong,
		Example: generateKeyExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, err := f.Validate()
			if err != nil {
				return err
			}

			resp, err := handler(req)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintGenerateKeyResponse(w, req, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"The wallet where the key is generated in",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)
	cmd.Flags().StringSliceVarP(&f.RawMetadata,
		"meta", "m",
		[]string{},
		`Metadata to add to the generated key-pair: "my-key1:my-value1,my-key2:my-value2"`,
	)

	autoCompleteWallet(cmd, rf.Home)

	return cmd
}

type GenerateKeyFlags struct {
	Wallet         string
	PassphraseFile string
	RawMetadata    []string
}

func (f *GenerateKeyFlags) Validate() (*wallet.GenerateKeyRequest, error) {
	req := &wallet.GenerateKeyRequest{}

	if len(f.Wallet) == 0 {
		return nil, flags.FlagMustBeSpecifiedError("wallet")
	}
	req.Wallet = f.Wallet

	metadata, err := cli.ParseMetadata(f.RawMetadata)
	if err != nil {
		return nil, err
	}
	req.Metadata = metadata

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return nil, err
	}
	req.Passphrase = passphrase

	return req, nil
}

func PrintGenerateKeyResponse(w io.Writer, req *wallet.GenerateKeyRequest, resp *wallet.GenerateKeyResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().Text("Key pair has been generated in wallet ").Bold(req.Wallet).NextLine()
	str.CheckMark().SuccessText("Generating a key pair succeeded").NextSection()
	str.Text("Public key:").NextLine()
	str.WarningText(resp.PublicKey).NextLine()
	str.Text("Metadata:").NextLine()
	printMeta(str, resp.Meta)
	str.NextSection()
	str.BlueArrow().InfoText("Run the service").NextLine()
	str.Text("Now, you can run the service. See the following command:").NextSection()
	str.Code(fmt.Sprintf("%s service run --help", os.Args[0])).NextLine()
}
