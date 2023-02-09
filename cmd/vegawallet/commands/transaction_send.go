package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	vgzap "code.vegaprotocol.io/vega/libs/zap"
	"code.vegaprotocol.io/vega/paths"
	coreversion "code.vegaprotocol.io/vega/version"
	"code.vegaprotocol.io/vega/wallet/api"
	walletnode "code.vegaprotocol.io/vega/wallet/api/node"
	networkStore "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"code.vegaprotocol.io/vega/wallet/version"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	sendTransactionLong = cli.LongDesc(`
		Send a transaction to a Vega node via the gRPC API. The transaction can be sent to
		any node of a registered network or to a specific node address.

		The transaction should be a Vega transaction formatted as a JSON payload, as follows:

		'{"commandName": {"someProperty": "someValue"} }'

		For vote submission, it will look like this:

		'{"voteSubmission": {"proposalId": "some-id", "value": "VALUE_YES"}}'
	`)

	sendTransactionExample = cli.Examples(`
		# Send a transaction to a registered network
		{{.Software}} transaction send --network NETWORK --wallet WALLET --pubkey PUBKEY TRANSACTION

		# Send a transaction to a specific Vega node address
		{{.Software}} transaction send --node-address ADDRESS --wallet WALLET --pubkey PUBKEY TRANSACTION

		# Send a transaction with a log level set to debug
		{{.Software}} transaction send --network NETWORK --wallet WALLET --pubkey PUBKEY --level debug TRANSACTION

		# Send a transaction with a maximum of 10 retries
		{{.Software}} transaction send --network NETWORK --wallet WALLET --pubkey PUBKEY --retries 10 TRANSACTION

		# Send a transaction to a registered network without verifying network version compatibility
		{{.Software}} transaction send --network NETWORK --wallet WALLET --pubkey PUBKEY --no-version-check TRANSACTION
	`)
)

type SendTransactionHandler func(api.AdminSendTransactionParams, *zap.Logger) (api.AdminSendTransactionResult, error)

func NewCmdSendTransaction(w io.Writer, rf *RootFlags) *cobra.Command {
	handler := func(params api.AdminSendTransactionParams, log *zap.Logger) (api.AdminSendTransactionResult, error) {
		vegaPaths := paths.New(rf.Home)

		walletStore, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return api.AdminSendTransactionResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		ns, err := networkStore.InitialiseStore(vegaPaths)
		if err != nil {
			return api.AdminSendTransactionResult{}, fmt.Errorf("couldn't initialise network store: %w", err)
		}

		sendTx := api.NewAdminSendTransaction(walletStore, ns, func(hosts []string, retries uint64) (walletnode.Selector, error) {
			return walletnode.BuildRoundRobinSelectorWithRetryingNodes(log, hosts, retries)
		})

		rawResult, errDetails := sendTx.Handle(context.Background(), params)
		if errDetails != nil {
			return api.AdminSendTransactionResult{}, errors.New(errDetails.Data)
		}
		return rawResult.(api.AdminSendTransactionResult), nil
	}

	return BuildCmdSendTransaction(w, handler, rf)
}

func BuildCmdSendTransaction(w io.Writer, handler SendTransactionHandler, rf *RootFlags) *cobra.Command {
	f := &SendTransactionFlags{}

	cmd := &cobra.Command{
		Use:     "send",
		Short:   "Send a transaction to a Vega node",
		Long:    sendTransactionLong,
		Example: sendTransactionExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if aLen := len(args); aLen == 0 {
				return flags.ArgMustBeSpecifiedError("transaction")
			} else if aLen > 1 {
				return flags.TooManyArgsError("transaction")
			}
			f.RawTransaction = args[0]

			req, err := f.Validate()
			if err != nil {
				return err
			}

			log, err := buildCmdLogger(rf.Output, f.LogLevel)
			if err != nil {
				return fmt.Errorf("failed to build a logger: %w", err)
			}

			resp, err := handler(req, log)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintSendTransactionResponse(w, resp, rf)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Network,
		"network", "n",
		"",
		"Network to which the transaction is sent",
	)
	cmd.Flags().StringVar(&f.NodeAddress,
		"node-address",
		"",
		"Vega node address to which the transaction is sent",
	)
	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Wallet holding the public key",
	)
	cmd.Flags().StringVarP(&f.PubKey,
		"pubkey", "k",
		"",
		"Public key of the key pair to use for signing (hex-encoded)",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)
	cmd.Flags().StringVar(&f.LogLevel,
		"level",
		zapcore.InfoLevel.String(),
		fmt.Sprintf("Set the log level: %v", vgzap.SupportedLogLevels),
	)
	cmd.Flags().Uint64Var(&f.Retries,
		"retries",
		DefaultForwarderRetryCount,
		"Number of retries when contacting the Vega node",
	)
	cmd.Flags().BoolVar(&f.NoVersionCheck,
		"no-version-check",
		false,
		"Do not check for network version compatibility",
	)

	autoCompleteNetwork(cmd, rf.Home)
	autoCompleteWallet(cmd, rf.Home, "wallet")
	autoCompleteLogLevel(cmd)

	return cmd
}

type SendTransactionFlags struct {
	Network        string
	NodeAddress    string
	Wallet         string
	PubKey         string
	PassphraseFile string
	Retries        uint64
	LogLevel       string
	RawTransaction string
	NoVersionCheck bool
}

func (f *SendTransactionFlags) Validate() (api.AdminSendTransactionParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminSendTransactionParams{}, flags.MustBeSpecifiedError("wallet")
	}

	if len(f.LogLevel) == 0 {
		return api.AdminSendTransactionParams{}, flags.MustBeSpecifiedError("level")
	}
	if err := vgzap.EnsureIsSupportedLogLevel(f.LogLevel); err != nil {
		return api.AdminSendTransactionParams{}, err
	}

	if len(f.NodeAddress) == 0 && len(f.Network) == 0 {
		return api.AdminSendTransactionParams{}, flags.OneOfFlagsMustBeSpecifiedError("network", "node-address")
	}

	if len(f.NodeAddress) != 0 && len(f.Network) != 0 {
		return api.AdminSendTransactionParams{}, flags.MutuallyExclusiveError("network", "node-address")
	}

	if len(f.PubKey) == 0 {
		return api.AdminSendTransactionParams{}, flags.MustBeSpecifiedError("pubkey")
	}

	if len(f.RawTransaction) == 0 {
		return api.AdminSendTransactionParams{}, flags.ArgMustBeSpecifiedError("transaction")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminSendTransactionParams{}, err
	}

	// Encode transaction into nested structure; this is a bit nasty but mirroring what happens
	// when our json-rpc library parses a request. There's an issue (6983#) to make the use
	// json.RawMessage instead.
	transaction := make(map[string]any)
	if err := json.Unmarshal([]byte(f.RawTransaction), &transaction); err != nil {
		return api.AdminSendTransactionParams{}, fmt.Errorf("couldn't unmarshal transaction: %w", err)
	}

	params := api.AdminSendTransactionParams{
		Wallet:      f.Wallet,
		Passphrase:  passphrase,
		PublicKey:   f.PubKey,
		Network:     f.Network,
		NodeAddress: f.NodeAddress,
		Retries:     f.Retries,
		Transaction: transaction,
		SendingMode: "TYPE_ASYNC",
	}

	return params, nil
}

func PrintSendTransactionResponse(w io.Writer, res api.AdminSendTransactionResult, rf *RootFlags) {
	p := printer.NewInteractivePrinter(w)

	if rf.Output == flags.InteractiveOutput && version.IsUnreleased() {
		str := p.String()
		str.CrossMark().DangerText("You are running an unreleased version of the Vega wallet (").DangerText(coreversion.Get()).DangerText(").").NextLine()
		str.Pad().DangerText("Use it at your own risk!").NextSection()
		p.Print(str)
	}

	str := p.String()
	defer p.Print(str)
	str.CheckMark().SuccessText("Transaction sending successful").NextSection()
	str.Text("Transaction Hash:").NextLine().WarningText(res.TxHash).NextSection()
	str.Text("Sent at:").NextLine().WarningText(res.SentAt.Format(time.ANSIC)).NextSection()
	str.Text("Selected node:").NextLine().WarningText(res.Node.Host).NextLine()
}
