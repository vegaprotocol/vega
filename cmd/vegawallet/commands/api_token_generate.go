package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/api"
	tokenStore "code.vegaprotocol.io/vega/wallet/api/session/store/v1"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"github.com/spf13/cobra"
)

var (
	generateAPITokenLong = cli.LongDesc(`
		Generate a long-living API token
	`)

	generateAPITokenExample = cli.Examples(`
		# Generate a long-living API token
		{{.Software}} api-token generate --description DESCRIPTION --wallet-name WALLET
	`)
)

type GenerateAPITokenHandler func(f GenerateAPITokenFlags, params api.AdminGenerateAPITokenParams) (api.AdminGenerateAPITokenResult, error)

func NewCmdGenerateAPIToken(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(f GenerateAPITokenFlags, params api.AdminGenerateAPITokenParams) (api.AdminGenerateAPITokenResult, error) {
		vegaPaths := paths.New(rf.Home)

		walletStore, err := wallets.InitialiseStoreFromPaths(vegaPaths)
		if err != nil {
			return api.AdminGenerateAPITokenResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		tokStore, err := tokenStore.LoadStore(vegaPaths, f.passphrase)
		if err != nil {
			if errors.Is(err, api.ErrWrongPassphrase) {
				return api.AdminGenerateAPITokenResult{}, err
			}
			return api.AdminGenerateAPITokenResult{}, fmt.Errorf("couldn't load the tokens store: %w", err)
		}

		generateAPIToken := api.NewAdminGenerateAPIToken(walletStore, tokStore)
		rawResult, errorDetails := generateAPIToken.Handle(context.Background(), params, jsonrpc.RequestMetadata{})
		if errorDetails != nil {
			return api.AdminGenerateAPITokenResult{}, errors.New(errorDetails.Data)
		}
		return rawResult.(api.AdminGenerateAPITokenResult), nil
	}

	return BuildCmdGenerateAPIToken(w, ensureAPITokensStoreIsInit, h, rf)
}

func BuildCmdGenerateAPIToken(w io.Writer, preCheck APITokePreCheck, handler GenerateAPITokenHandler, rf *RootFlags) *cobra.Command {
	f := &GenerateAPITokenFlags{}

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate a long-living API token",
		Long:    generateAPITokenLong,
		Example: generateAPITokenExample,
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
				printGeneratedAPIToken(w, params, res)
			case flags.JSONOutput:
				return printer.FprintJSON(w, res)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&f.Description,
		"description",
		"",
		"Description of the token purpose",
	)

	cmd.Flags().StringVar(&f.PassphraseFile,
		"tokens-passphrase-file",
		"",
		"Path to the file containing the tokens database passphrase",
	)

	cmd.Flags().StringVar(&f.WalletName,
		"wallet-name",
		"",
		"Name of the wallet associated to the token",
	)

	cmd.Flags().StringVar(&f.WalletPassphraseFile,
		"wallet-passphrase-file",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	cmd.Flags().DurationVar(&f.ExpiresIn,
		"expires-in",
		0,
		"How duration for which the token will be valid",
	)

	autoCompleteWallet(cmd, f.WalletName, "wallet-name")

	return cmd
}

type GenerateAPITokenFlags struct {
	Description          string
	PassphraseFile       string
	WalletName           string
	WalletPassphraseFile string
	ExpiresIn            time.Duration
	passphrase           string
}

func (f *GenerateAPITokenFlags) Validate() (api.AdminGenerateAPITokenParams, error) {
	if len(f.WalletName) == 0 {
		return api.AdminGenerateAPITokenParams{}, flags.MustBeSpecifiedError("wallet-name")
	}

	passphrase, err := flags.GetPassphraseWithOptions(flags.PassphraseOptions{Name: "tokens"}, f.PassphraseFile)
	if err != nil {
		return api.AdminGenerateAPITokenParams{}, err
	}
	f.passphrase = passphrase

	walletPassphrase, err := flags.GetPassphraseWithOptions(flags.PassphraseOptions{Name: "wallet"}, f.WalletPassphraseFile)
	if err != nil {
		return api.AdminGenerateAPITokenParams{}, err
	}

	var expiry *int64
	if f.ExpiresIn != 0 {
		expiry = ptr.From(time.Now().Add(f.ExpiresIn).Unix())
	}

	tokenParams := api.AdminGenerateAPITokenParams{
		Description: f.Description,
		ExpiryTs:    expiry,
		Wallet: api.AdminGenerateAPITokenWalletParams{
			Name:       f.WalletName,
			Passphrase: walletPassphrase,
		},
	}
	params := tokenParams
	return params, nil
}

func printGeneratedAPIToken(w io.Writer, params api.AdminGenerateAPITokenParams, res api.AdminGenerateAPITokenResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().Text("The API token has been successfully generated: ").SuccessText(res.Token).NextSection()

	str.RedArrow().DangerText("Important").NextLine()
	str.DangerText("This token can be used by third-party applications to access the wallet ").DangerBold(params.Wallet.Name).DangerText(" and send transactions from it, automatically, without human intervention!").NextLine()
	str.DangerBold("Only distribute it to applications you have absolute trust in.").NextLine()
}
