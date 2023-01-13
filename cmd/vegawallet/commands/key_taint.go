package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	taintKeyLong = cli.LongDesc(`
		Tainting a key pair marks it as unsafe to use and ensures it will not be
		used to sign transactions.

		This mechanism is useful when the key pair has been compromised.

		When a key is tainted, it is automatically removed from the allowed
		keys if specified. If the key is the only one to be set, the permission
		to access the public keys is revoked. If no allowed key is specified,
		but all keys in the wallet are tainted, the permission of the public
		keys is revoked as well.
	`)

	taintKeyExample = cli.Examples(`
		# Taint a key pair
		{{.Software}} key taint --wallet WALLET --pubkey PUBKEY
	`)
)

type TaintKeyHandler func(api.AdminTaintKeyParams) error

func NewCmdTaintKey(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminTaintKeyParams) error {
		walletStore, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		taintKey := api.NewAdminTaintKey(walletStore)
		_, errDetails := taintKey.Handle(context.Background(), params)
		if errDetails != nil {
			return errors.New(errDetails.Data)
		}
		return nil
	}

	return BuildCmdTaintKey(w, h, rf)
}

func BuildCmdTaintKey(w io.Writer, handler TaintKeyHandler, rf *RootFlags) *cobra.Command {
	f := &TaintKeyFlags{}

	cmd := &cobra.Command{
		Use:     "taint",
		Short:   "Mark a key pair as tainted",
		Long:    taintKeyLong,
		Example: taintKeyExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, err := f.Validate()
			if err != nil {
				return err
			}

			if err := handler(req); err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintTaintKeyResponse(w)
			case flags.JSONOutput:
				return nil
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Wallet holding the public key",
	)
	cmd.Flags().StringVarP(&f.PublicKey,
		"pubkey", "k",
		"",
		"Public key to taint (hex-encoded)",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type TaintKeyFlags struct {
	Wallet         string
	PublicKey      string
	PassphraseFile string
}

func (f *TaintKeyFlags) Validate() (api.AdminTaintKeyParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminTaintKeyParams{}, flags.MustBeSpecifiedError("wallet")
	}

	if len(f.PublicKey) == 0 {
		return api.AdminTaintKeyParams{}, flags.MustBeSpecifiedError("pubkey")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminTaintKeyParams{}, err
	}

	return api.AdminTaintKeyParams{
		Wallet:     f.Wallet,
		PublicKey:  f.PublicKey,
		Passphrase: passphrase,
	}, nil
}

func PrintTaintKeyResponse(w io.Writer) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("Tainting succeeded").NextSection()

	str.RedArrow().DangerText("Important").NextLine()
	str.Text("If you tainted a key for security reasons, you should not untaint it.").NextSection()

	str.BlueArrow().InfoText("Untaint a key").NextLine()
	str.Text("You may have tainted a key pair by mistake. If you want to untaint it, see the following command:").NextSection()
	str.Code(fmt.Sprintf("%s key untaint --help", os.Args[0])).NextLine()
}
