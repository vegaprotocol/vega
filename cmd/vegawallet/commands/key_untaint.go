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
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	untaintKeyLong = cli.LongDesc(`
		Remove the taint from a key pair.

		If you tainted a key for security reasons, you should not untaint it.
	`)

	untaintKeyExample = cli.Examples(`
		# Untaint a key pair
		{{.Software}} key untaint --wallet WALLET --pubkey PUBKEY
	`)
)

type UntaintKeyHandler func(api.AdminUntaintKeyParams) error

func NewCmdUntaintKey(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminUntaintKeyParams) error {
		s, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		untaintKey := api.NewAdminUntaintKey(s)
		_, errDetails := untaintKey.Handle(context.Background(), params, jsonrpc.RequestMetadata{})
		if errDetails != nil {
			return errors.New(errDetails.Data)
		}
		return nil
	}

	return BuildCmdUntaintKey(w, h, rf)
}

func BuildCmdUntaintKey(w io.Writer, handler UntaintKeyHandler, rf *RootFlags) *cobra.Command {
	f := &UntaintKeyFlags{}

	cmd := &cobra.Command{
		Use:     "untaint",
		Short:   "Remove the taint from a key pair",
		Long:    untaintKeyLong,
		Example: untaintKeyExample,
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
				PrintUntaintKeyResponse(w)
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
		"Public key to untaint (hex-encoded)",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type UntaintKeyFlags struct {
	Wallet         string
	PublicKey      string
	PassphraseFile string
}

func (f *UntaintKeyFlags) Validate() (api.AdminUntaintKeyParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminUntaintKeyParams{}, flags.MustBeSpecifiedError("wallet")
	}

	if len(f.PublicKey) == 0 {
		return api.AdminUntaintKeyParams{}, flags.MustBeSpecifiedError("pubkey")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminUntaintKeyParams{}, err
	}

	return api.AdminUntaintKeyParams{
		Wallet:     f.Wallet,
		PublicKey:  f.PublicKey,
		Passphrase: passphrase,
	}, nil
}

func PrintUntaintKeyResponse(w io.Writer) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.CheckMark().SuccessText("Untainting succeeded").NextSection()

	str.RedArrow().DangerText("Important").NextLine()
	str.Text("If you tainted a key for security reasons, you should not use it.").NextLine()

	str.BlueArrow().InfoText("Taint a key").NextLine()
	str.Text("To taint a key pair, see the following command:").NextSection()
	str.Code(fmt.Sprintf("%s key taint --help", os.Args[0])).NextLine()
}
