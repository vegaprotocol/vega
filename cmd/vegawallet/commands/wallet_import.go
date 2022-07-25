package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"code.vegaprotocol.io/vega/wallet/wallets"
	"github.com/spf13/cobra"
)

var (
	importWalletLong = cli.LongDesc(`
		Import a wallet using the recovery phrase and generate the first Ed25519 key pair.

		You will be asked to create a passphrase. The passphrase is used to protect
		the file in which the keys are stored. Hence, it can be different from the
		original passphrase, used during the wallet creation. This doesn't affect the
		key generation process in any way.
	`)

	importWalletExample = cli.Examples(`
		# Import a wallet using the recovery phrase
		{{.Software}} import --wallet WALLET --recovery-phrase-file PATH_TO_RECOVERY_PHRASE

		# Import an older version of the wallet using the recovery phrase
		{{.Software}} import --wallet WALLET --recovery-phrase-file PATH_TO_RECOVERY_PHRASE --version VERSION
	`)
)

type ImportWalletHandler func(*wallet.ImportWalletRequest) (*wallet.ImportWalletResponse, error)

func NewCmdImportWallet(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(req *wallet.ImportWalletRequest) (*wallet.ImportWalletResponse, error) {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		return wallet.ImportWallet(s, req)
	}

	return BuildCmdImportWallet(w, h, rf)
}

func BuildCmdImportWallet(w io.Writer, handler ImportWalletHandler, rf *RootFlags) *cobra.Command {
	f := &ImportWalletFlags{}

	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Import a wallet using the recovery phrase",
		Long:    importWalletLong,
		Example: importWalletExample,
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
				PrintImportWalletResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Name of the wallet to use",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the passphrase to access the wallet",
	)
	cmd.Flags().StringVar(&f.RecoveryPhraseFile,
		"recovery-phrase-file",
		"",
		"Path to the file containing the recovery phrase of the wallet",
	)
	cmd.Flags().Uint32Var(&f.Version,
		"version",
		wallet.LatestVersion,
		fmt.Sprintf("Version of the wallet to import: %v", wallet.SupportedVersions),
	)

	_ = cmd.RegisterFlagCompletionFunc("version", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		vs := make([]string, 0, len(wallet.SupportedVersions))
		for i, v := range wallet.SupportedVersions {
			vs[i] = strconv.FormatUint(uint64(v), 10) //nolint:gomnd
		}
		return SupportedLogLevels, cobra.ShellCompDirectiveDefault
	})

	return cmd
}

type ImportWalletFlags struct {
	Wallet             string
	PassphraseFile     string
	RecoveryPhraseFile string
	Version            uint32
}

func (f *ImportWalletFlags) Validate() (*wallet.ImportWalletRequest, error) {
	req := &wallet.ImportWalletRequest{
		Version: f.Version,
	}

	if len(f.Wallet) == 0 {
		return nil, flags.FlagMustBeSpecifiedError("wallet")
	}
	req.Wallet = f.Wallet

	if len(f.RecoveryPhraseFile) == 0 {
		return nil, flags.FlagMustBeSpecifiedError("recovery-phrase-file")
	}
	recoveryPhrase, err := vgfs.ReadFile(f.RecoveryPhraseFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't read recovery phrase file: %w", err)
	}
	req.RecoveryPhrase = strings.Trim(string(recoveryPhrase), "\n")

	passphrase, err := flags.GetConfirmedPassphrase(f.PassphraseFile)
	if err != nil {
		return nil, err
	}
	req.Passphrase = passphrase

	return req, nil
}

func PrintImportWalletResponse(w io.Writer, resp *wallet.ImportWalletResponse) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().Text("Wallet ").Bold(resp.Wallet.Name).Text(" has been imported at: ").SuccessText(resp.Wallet.FilePath).NextLine()
	str.CheckMark().Text("First key pair has been generated for wallet ").Bold(resp.Wallet.Name).Text(" at: ").SuccessText(resp.Wallet.FilePath).NextLine()
	str.CheckMark().SuccessText("Importing the wallet succeeded").NextSection()

	str.Text("First public key:").NextLine()
	str.WarningText(resp.Key.PublicKey).NextLine()
	str.NextSection()

	str.BlueArrow().InfoText("Run the service").NextLine()
	str.Text("Now, you can run the service. See the following command:").NextSection()
	str.Code(fmt.Sprintf("%s service run --help", os.Args[0])).NextLine()
}
