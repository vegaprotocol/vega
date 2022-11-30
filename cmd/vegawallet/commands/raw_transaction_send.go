package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	vgzap "code.vegaprotocol.io/vega/libs/zap"
	"code.vegaprotocol.io/vega/paths"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	walletnode "code.vegaprotocol.io/vega/wallet/api/node"
	networkStore "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"github.com/golang/protobuf/proto"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	sendTxLong = cli.LongDesc(`
		Send a transaction to a Vega node via the gRPC API. The command can be sent to
		any node of a registered network or to a specific node address.

		The transaction should be base64-encoded.
	`)

	sendTxExample = cli.Examples(`
		# Send a transaction to a registered network
		{{.Software}} tx send --network NETWORK BASE64_TRANSACTION

		# Send a transaction to a specific Vega node address
		{{.Software}} tx send --node-address ADDRESS BASE64_TRANSACTION

		# Send a transaction with a log level set to debug
		{{.Software}} tx send --network NETWORK --level debug BASE64_TRANSACTION

		# Send a transaction with a maximum of 10 retries
		{{.Software}} tx send --network NETWORK --retries 10 BASE64_TRANSACTION

		# Send a transaction without verifying network version compatibility
		{{.Software}} tx send --network NETWORK --retries 10 BASE64_TRANSACTION --no-version-check
	`)
)

type SendTxHandler func(api.AdminSendTransactionParams, *zap.Logger) (api.AdminSendTransactionResult, error)

func NewCmdTxSend(w io.Writer, rf *RootFlags) *cobra.Command {
	h := func(params api.AdminSendTransactionParams, log *zap.Logger) (api.AdminSendTransactionResult, error) {
		vegaPaths := paths.New(rf.Home)

		netStore, err := networkStore.InitialiseStore(vegaPaths)
		if err != nil {
			return api.AdminSendTransactionResult{}, fmt.Errorf("couldn't initialise network store: %w", err)
		}

		sendTransaction := api.NewAdminSendTransaction(netStore, func(hosts []string, retries uint64) (walletnode.Selector, error) {
			return walletnode.BuildRoundRobinSelectorWithRetryingNodes(log, hosts, retries)
		})
		rawResult, errorDetails := sendTransaction.Handle(context.Background(), params)
		if errorDetails != nil {
			return api.AdminSendTransactionResult{}, errors.New(errorDetails.Data)
		}
		return rawResult.(api.AdminSendTransactionResult), nil
	}
	return BuildCmdTxSend(w, h, rf)
}

func BuildCmdTxSend(w io.Writer, handler SendTxHandler, rf *RootFlags) *cobra.Command {
	f := &SendTxFlags{}

	cmd := &cobra.Command{
		Use:     "send",
		Short:   "Send a transaction to a Vega node",
		Long:    sendTxLong,
		Example: sendTxExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if aLen := len(args); aLen == 0 {
				return flags.ArgMustBeSpecifiedError("transaction")
			} else if aLen > 1 {
				return flags.TooManyArgsError("transaction")
			}
			f.RawTx = args[0]

			req, err := f.Validate()
			if err != nil {
				return err
			}

			log, err := buildCmdLogger(rf.Output, f.LogLevel)
			if err != nil {
				return fmt.Errorf("couldn't initialise logger: %w", err)
			}
			defer vgzap.Sync(log)

			resp, err := handler(req, log)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintTXSendResponse(w, resp)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Network,
		"network", "n",
		"",
		"Network to which the command is sent",
	)
	cmd.Flags().StringVar(&f.NodeAddress,
		"node-address",
		"",
		"Vega node address to which the command is sent",
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
	autoCompleteLogLevel(cmd)
	return cmd
}

type SendTxFlags struct {
	Network        string
	NodeAddress    string
	Retries        uint64
	LogLevel       string
	RawTx          string
	NoVersionCheck bool
}

func (f *SendTxFlags) Validate() (api.AdminSendTransactionParams, error) {
	req := api.AdminSendTransactionParams{
		Retries: f.Retries,
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
	req.NodeAddress = f.NodeAddress
	req.Network = f.Network
	req.SendingMode = "TYPE_ASYNC"

	if len(f.RawTx) == 0 {
		return api.AdminSendTransactionParams{}, flags.ArgMustBeSpecifiedError("transaction")
	}
	decodedTx, err := base64.StdEncoding.DecodeString(f.RawTx)
	if err != nil {
		return api.AdminSendTransactionParams{}, flags.MustBase64EncodedError("transaction")
	}
	tx := &commandspb.Transaction{}
	if err := proto.Unmarshal(decodedTx, tx); err != nil {
		return api.AdminSendTransactionParams{}, fmt.Errorf("couldn't unmarshal transaction: %w", err)
	}
	req.EncodedTransaction = f.RawTx

	return req, nil
}

func PrintTXSendResponse(w io.Writer, resp api.AdminSendTransactionResult) {
	p := printer.NewInteractivePrinter(w)

	str := p.String()
	defer p.Print(str)
	str.CheckMark().SuccessText("Transaction successfully sent").NextSection()
	str.Text("Transaction Hash:").NextLine().WarningText(resp.TxHash).NextSection()
}
