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
	checkTransactionLong = cli.LongDesc(`
		Check a transaction via the gRPC API. The transaction can be sent to
		any node of a registered network or to a specific node address.

		The transaction should be a Vega transaction formatted as a JSON payload, as follows:

		'{"commandName": {"someProperty": "someValue"} }'

		For vote submission, it will look like this:

		'{"voteSubmission": {"proposalId": "some-id", "value": "VALUE_YES"}}'
	`)

	checkTransactionExample = cli.Examples(`
		# Check a transaction and send it to a registered network
		{{.Software}} transaction check --network NETWORK --wallet WALLET --pubkey PUBKEY TRANSACTION

		# Check a transaction and send it to a specific Vega node address
		{{.Software}} transaction check --node-address ADDRESS --wallet WALLET --pubkey PUBKEY TRANSACTION

		# Check a transaction with a log level set to debug
		{{.Software}} transaction check --network NETWORK --wallet WALLET --pubkey PUBKEY --level debug TRANSACTION

		# Check a transaction with a maximum of 10 retries
		{{.Software}} transaction check --network NETWORK --wallet WALLET --pubkey PUBKEY --retries 10 TRANSACTION

		# Check a transaction and send it to a registered network without verifying network version compatibility
		{{.Software}} transaction check --network NETWORK --wallet WALLET --pubkey PUBKEY --no-version-check TRANSACTION
	`)
)

type CheckTransactionHandler func(api.AdminCheckTransactionParams, *zap.Logger) (api.AdminCheckTransactionResult, error)

func NewCmdCheckTransaction(w io.Writer, rf *RootFlags) *cobra.Command {
	handler := func(params api.AdminCheckTransactionParams, log *zap.Logger) (api.AdminCheckTransactionResult, error) {
		vegaPaths := paths.New(rf.Home)

		walletStore, err := wallets.InitialiseStore(rf.Home, false)
		if err != nil {
			return api.AdminCheckTransactionResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}
		defer walletStore.Close()

		ns, err := networkStore.InitialiseStore(vegaPaths)
		if err != nil {
			return api.AdminCheckTransactionResult{}, fmt.Errorf("couldn't initialise network store: %w", err)
		}

		checkTx := api.NewAdminCheckTransaction(walletStore, ns, func(hosts []string, retries uint64) (walletnode.Selector, error) {
			return walletnode.BuildRoundRobinSelectorWithRetryingNodes(log, hosts, retries)
		})

		rawResult, errDetails := checkTx.Handle(context.Background(), params)
		if errDetails != nil {
			return api.AdminCheckTransactionResult{}, errors.New(errDetails.Data)
		}
		return rawResult.(api.AdminCheckTransactionResult), nil
	}

	return BuildCmdCheckTransaction(w, handler, rf)
}

func BuildCmdCheckTransaction(w io.Writer, handler CheckTransactionHandler, rf *RootFlags) *cobra.Command {
	f := &CheckTransactionFlags{}

	cmd := &cobra.Command{
		Use:     "check",
		Short:   "Check a transaction and send it to a Vega node",
		Long:    checkTransactionLong,
		Example: checkTransactionExample,
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
				PrintCheckTransactionResponse(w, resp, rf)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Network,
		"network", "n",
		"",
		"Network to send the transaction to after it is checked",
	)
	cmd.Flags().StringVar(&f.NodeAddress,
		"node-address",
		"",
		"Vega node address to which the transaction is sent after it is checked",
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

type CheckTransactionFlags struct {
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

func (f *CheckTransactionFlags) Validate() (api.AdminCheckTransactionParams, error) {
	if len(f.Wallet) == 0 {
		return api.AdminCheckTransactionParams{}, flags.MustBeSpecifiedError("wallet")
	}

	if len(f.LogLevel) == 0 {
		return api.AdminCheckTransactionParams{}, flags.MustBeSpecifiedError("level")
	}
	if err := vgzap.EnsureIsSupportedLogLevel(f.LogLevel); err != nil {
		return api.AdminCheckTransactionParams{}, err
	}

	if len(f.NodeAddress) == 0 && len(f.Network) == 0 {
		return api.AdminCheckTransactionParams{}, flags.OneOfFlagsMustBeSpecifiedError("network", "node-address")
	}

	if len(f.NodeAddress) != 0 && len(f.Network) != 0 {
		return api.AdminCheckTransactionParams{}, flags.MutuallyExclusiveError("network", "node-address")
	}

	if len(f.PubKey) == 0 {
		return api.AdminCheckTransactionParams{}, flags.MustBeSpecifiedError("pubkey")
	}

	if len(f.RawTransaction) == 0 {
		return api.AdminCheckTransactionParams{}, flags.ArgMustBeSpecifiedError("transaction")
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminCheckTransactionParams{}, err
	}

	// Encode transaction into a nested structure; this is a bit nasty but mirroring what happens
	// when our json-rpc library parses a request. There's an issue (6983#) to make the use
	// json.RawMessage instead.
	transaction := make(map[string]any)
	if err := json.Unmarshal([]byte(f.RawTransaction), &transaction); err != nil {
		return api.AdminCheckTransactionParams{}, fmt.Errorf("could not unmarshal transaction: %w", err)
	}

	params := api.AdminCheckTransactionParams{
		Wallet:      f.Wallet,
		Passphrase:  passphrase,
		PublicKey:   f.PubKey,
		Network:     f.Network,
		NodeAddress: f.NodeAddress,
		Retries:     f.Retries,
		Transaction: transaction,
	}

	return params, nil
}

func PrintCheckTransactionResponse(w io.Writer, res api.AdminCheckTransactionResult, rf *RootFlags) {
	p := printer.NewInteractivePrinter(w)

	if rf.Output == flags.InteractiveOutput && version.IsUnreleased() {
		str := p.String()
		str.CrossMark().DangerText("You are running an unreleased version of the Vega wallet (").DangerText(coreversion.Get()).DangerText(").").NextLine()
		str.Pad().DangerText("Use it at your own risk!").NextSection()
		p.Print(str)
	}

	str := p.String()
	defer p.Print(str)
	str.CheckMark().SuccessText("Transaction checking successful").NextSection()
	str.Text("Sent at:").NextLine().WarningText(res.SentAt.Format(time.ANSIC)).NextSection()
	str.Text("Selected node:").NextLine().WarningText(res.Node.Host).NextLine()
}
