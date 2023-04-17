package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
)

var (
	describeKeyLong = cli.LongDesc(`
		Describe all known information about the specified key pair
	`)

	describeKeyExample = cli.Examples(`
		# Describe a key
		{{.Software}} key describe --wallet WALLET --pubkey PUBLIC_KEY
	`)
)

type DescribeKeyHandler func(api.AdminDescribeKeyParams) (api.AdminDescribeKeyResult, error)

func NewCmdDescribeKey(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminDescribeKeyParams) (api.AdminDescribeKeyResult, error) {
		walletStore, err := wallets.InitialiseStore(rf.Home, false)
		if err != nil {
			return api.AdminDescribeKeyResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		describeKey := api.NewAdminDescribeKey(walletStore)
		rawResult, errDetails := describeKey.Handle(context.Background(), params)
		if errDetails != nil {
			return api.AdminDescribeKeyResult{}, errors.New(errDetails.Data)
		}
		return rawResult.(api.AdminDescribeKeyResult), nil
	}

	return BuildCmdDescribeKey(w, h, rf)
}

func BuildCmdDescribeKey(w io.Writer, handler DescribeKeyHandler, rf *RootFlags) *cobra.Command {
	f := &DescribeKeyFlags{}

	cmd := &cobra.Command{
		Use:     "describe",
		Short:   "Describe the specified key pair",
		Long:    describeKeyLong,
		Example: describeKeyExample,
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
				PrintDescribeKeyResponse(w, resp)
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
	cmd.Flags().StringVarP(&f.PublicKey,
		"pubkey", "k",
		"",
		"Public key to describe (hex-encoded)",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type DescribeKeyFlags struct {
	Wallet         string
	PassphraseFile string
	PublicKey      string
}

func (f *DescribeKeyFlags) Validate() (api.AdminDescribeKeyParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminDescribeKeyParams{}, flags.MustBeSpecifiedError("wallet")
	}

	if len(f.PublicKey) == 0 {
		return api.AdminDescribeKeyParams{}, flags.MustBeSpecifiedError("pubkey")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminDescribeKeyParams{}, err
	}

	return api.AdminDescribeKeyParams{
		Wallet:     f.Wallet,
		Passphrase: passphrase,
		PublicKey:  f.PublicKey,
	}, nil
}

func PrintDescribeKeyResponse(w io.Writer, resp api.AdminDescribeKeyResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)

	str.Text("Name:              ").WarningText(resp.Name).NextLine()
	str.Text("Public key:        ").WarningText(resp.PublicKey).NextLine()
	str.Text("Algorithm Name:    ").WarningText(resp.Algorithm.Name).NextLine()
	str.Text("Algorithm Version: ").WarningText(fmt.Sprint(resp.Algorithm.Version)).NextSection()

	str.Text("Key pair is: ")
	switch resp.IsTainted {
	case true:
		str.DangerText("tainted").NextLine()
	case false:
		str.SuccessText("not tainted").NextLine()
	}
	str.Text("Tainting a key marks it as unsafe to use and ensures it will not be used to sign transactions.").NextLine()
	str.Text("This mechanism is useful when the key pair has been compromised.").NextSection()

	str.Text("Metadata:").NextLine()
	printMeta(str, resp.Metadata)
}
