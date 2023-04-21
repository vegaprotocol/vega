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
	annotateKeyLong = cli.LongDesc(`
		Add metadata to a key pair. All existing metadata is removed and replaced
		by the specified new metadata.

		The metadata is a list of key-value pairs. A key-value is colon-separated,
		and the key-values are comma-separated.

		It is possible to give a name to a key pair, that is recognised by user
		interfaces, by setting the metadata "name".
	`)

	annotateKeyExample = cli.Examples(`
		Given the following metadata to be added:
			- name: my-wallet
			- type: validation

		# Annotate a key pair
		{{.Software}} key annotate --wallet WALLET --pubkey PUBKEY --meta "name:my-wallet,type:validation"

		# Remove all metadata from a key pair
		{{.Software}} key annotate --wallet WALLET --pubkey PUBKEY --clear
	`)
)

type AnnotateKeyHandler func(api.AdminAnnotateKeyParams, string) (api.AdminAnnotateKeyResult, error)

func NewCmdAnnotateKey(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminAnnotateKeyParams, passphrase string) (api.AdminAnnotateKeyResult, error) {
		ctx := context.Background()

		walletStore, err := wallets.InitialiseStore(rf.Home, false)
		if err != nil {
			return api.AdminAnnotateKeyResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		if _, errDetails := api.NewAdminUnlockWallet(walletStore).Handle(ctx, api.AdminUnlockWalletParams{
			Wallet:     params.Wallet,
			Passphrase: passphrase,
		}); errDetails != nil {
			return api.AdminAnnotateKeyResult{}, errors.New(errDetails.Data)
		}

		rawResult, errDetails := api.NewAdminAnnotateKey(walletStore).Handle(ctx, params)
		if errDetails != nil {
			return api.AdminAnnotateKeyResult{}, errors.New(errDetails.Data)
		}
		return rawResult.(api.AdminAnnotateKeyResult), nil
	}

	return BuildCmdAnnotateKey(w, h, rf)
}

func BuildCmdAnnotateKey(w io.Writer, handler AnnotateKeyHandler, rf *RootFlags) *cobra.Command {
	f := AnnotateKeyFlags{}

	cmd := &cobra.Command{
		Use:     "annotate",
		Short:   "Add metadata to a key pair",
		Long:    annotateKeyLong,
		Example: annotateKeyExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			req, pass, err := f.Validate()
			if err != nil {
				return err
			}

			resp, err := handler(req, pass)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintAnnotateKeyResponse(w, f, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Wallet holding the public key",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)
	cmd.Flags().StringVarP(&f.PubKey,
		"pubkey", "k",
		"",
		"Public key to annotate (hex-encoded)",
	)
	cmd.Flags().StringSliceVarP(&f.RawMetadata,
		"meta", "m",
		[]string{},
		`A list of metadata e.g: "my-key1:my-value1,my-key2:my-value2"`,
	)
	cmd.Flags().BoolVar(&f.Clear,
		"clear",
		false,
		"Clear the metadata",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type AnnotateKeyFlags struct {
	Wallet         string
	PubKey         string
	PassphraseFile string
	Clear          bool
	RawMetadata    []string
}

func (f *AnnotateKeyFlags) Validate() (api.AdminAnnotateKeyParams, string, error) {
	if len(f.Wallet) == 0 {
		return api.AdminAnnotateKeyParams{}, "", flags.MustBeSpecifiedError("wallet")
	}

	if len(f.PubKey) == 0 {
		return api.AdminAnnotateKeyParams{}, "", flags.MustBeSpecifiedError("pubkey")
	}

	if len(f.RawMetadata) == 0 && !f.Clear {
		return api.AdminAnnotateKeyParams{}, "", flags.OneOfFlagsMustBeSpecifiedError("meta", "clear")
	}
	if len(f.RawMetadata) != 0 && f.Clear {
		return api.AdminAnnotateKeyParams{}, "", flags.MutuallyExclusiveError("meta", "clear")
	}

	metadata, err := cli.ParseMetadata(f.RawMetadata)
	if err != nil {
		return api.AdminAnnotateKeyParams{}, "", err
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminAnnotateKeyParams{}, "", err
	}

	return api.AdminAnnotateKeyParams{
		Wallet:    f.Wallet,
		PublicKey: f.PubKey,
		Metadata:  metadata,
	}, passphrase, nil
}

func PrintAnnotateKeyResponse(w io.Writer, f AnnotateKeyFlags, res api.AdminAnnotateKeyResult) {
	p := printer.NewInteractivePrinter(w)
	str := p.String()
	defer p.Print(str)
	if f.Clear {
		str.CheckMark().SuccessText("Annotation cleared").NextLine()
	} else {
		str.CheckMark().SuccessText("Annotation succeeded").NextSection()
	}
	str.Text("New metadata:").NextLine()
	printMeta(str, res.Metadata)
}
