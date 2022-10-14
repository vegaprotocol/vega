package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/libs/crypto"
	vglog "code.vegaprotocol.io/vega/libs/zap"
	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	coreversion "code.vegaprotocol.io/vega/version"
	wcommands "code.vegaprotocol.io/vega/wallet/commands"
	"code.vegaprotocol.io/vega/wallet/network"
	"code.vegaprotocol.io/vega/wallet/node"
	"code.vegaprotocol.io/vega/wallet/version"
	walletversion "code.vegaprotocol.io/vega/wallet/version"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/golang/protobuf/jsonpb"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrDoNotSetPubKeyInCommand = errors.New("do not set the public key through the command, use --pubkey flag instead")

	sendCommandLong = cli.LongDesc(`
		Send a command to a Vega node via the gRPC API. The command can be sent to
		any node of a registered network or to a specific node address.

		The command should be a Vega command formatted as a JSON payload, as follows:

		'{"commandName": {"someProperty": "someValue"} }'

		For vote submission, it will look like this:

		'{"voteSubmission": {"proposalId": "some-id", "value": "VALUE_YES"}}'
	`)

	sendCommandExample = cli.Examples(`
		# Send a command to a registered network
		{{.Software}} command send --network NETWORK --wallet WALLET --pubkey PUBKEY COMMAND

		# Send a command to a specific Vega node address
		{{.Software}} command send --node-address ADDRESS --wallet WALLET --pubkey PUBKEY COMMAND

		# Send a command with a log level set to debug
		{{.Software}} command send --network NETWORK --wallet WALLET --pubkey PUBKEY --level debug COMMAND

		# Send a command with a maximum of 10 retries
		{{.Software}} command send --network NETWORK --wallet WALLET --pubkey PUBKEY --retries 10 COMMAND

		# Send a command to a registered network without verifying network version compatibility
		{{.Software}} command send --network NETWORK --wallet WALLET --pubkey PUBKEY --no-version-check COMMAND
	`)
)

type SendCommandHandler func(io.Writer, *SendCommandFlags, *RootFlags, *SendCommandRequest) error

func NewCmdCommandSend(w io.Writer, rf *RootFlags) *cobra.Command {
	return BuildCmdCommandSend(w, SendCommand, rf)
}

func BuildCmdCommandSend(w io.Writer, handler SendCommandHandler, rf *RootFlags) *cobra.Command {
	f := &SendCommandFlags{}

	cmd := &cobra.Command{
		Use:     "send",
		Short:   "Send a command to a Vega node",
		Long:    sendCommandLong,
		Example: sendCommandExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if aLen := len(args); aLen == 0 {
				return flags.ArgMustBeSpecifiedError("command")
			} else if aLen > 1 {
				return flags.TooManyArgsError("command")
			}
			f.RawCommand = args[0]

			req, err := f.Validate()
			if err != nil {
				return err
			}

			if err := handler(w, f, rf, req); err != nil {
				return err
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
		fmt.Sprintf("Set the log level: %v", SupportedLogLevels),
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
	autoCompleteWallet(cmd, rf.Home)
	autoCompleteLogLevel(cmd)

	return cmd
}

type SendCommandFlags struct {
	Network        string
	NodeAddress    string
	Wallet         string
	PubKey         string
	PassphraseFile string
	Retries        uint64
	LogLevel       string
	RawCommand     string
	NoVersionCheck bool
}

func (f *SendCommandFlags) Validate() (*SendCommandRequest, error) {
	req := &SendCommandRequest{
		Retries: f.Retries,
	}

	if len(f.Wallet) == 0 {
		return nil, flags.MustBeSpecifiedError("wallet")
	}
	req.Wallet = f.Wallet

	if len(f.LogLevel) == 0 {
		return nil, flags.MustBeSpecifiedError("level")
	}
	if err := ValidateLogLevel(f.LogLevel); err != nil {
		return nil, err
	}
	req.LogLevel = f.LogLevel

	if len(f.NodeAddress) == 0 && len(f.Network) == 0 {
		return nil, flags.OneOfFlagsMustBeSpecifiedError("network", "node-address")
	}
	if len(f.NodeAddress) != 0 && len(f.Network) != 0 {
		return nil, flags.MutuallyExclusiveError("network", "node-address")
	}
	req.NodeAddress = f.NodeAddress
	req.Network = f.Network

	if len(f.PubKey) == 0 {
		return nil, flags.MustBeSpecifiedError("pubkey")
	}
	if len(f.RawCommand) == 0 {
		return nil, flags.ArgMustBeSpecifiedError("command")
	}
	request := &walletpb.SubmitTransactionRequest{}
	if err := jsonpb.UnmarshalString(f.RawCommand, request); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal command as request: %w", err)
	}
	if len(request.PubKey) != 0 {
		return nil, ErrDoNotSetPubKeyInCommand
	}
	request.PubKey = f.PubKey
	request.Propagate = true
	req.Request = request
	if errs := wcommands.CheckSubmitTransactionRequest(req.Request); !errs.Empty() {
		return nil, fmt.Errorf("invalid request: %w", errs)
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return nil, err
	}
	req.Passphrase = passphrase

	return req, nil
}

type SendCommandRequest struct {
	Network     string
	NodeAddress string
	Wallet      string
	Passphrase  string
	Retries     uint64
	LogLevel    string
	Request     *walletpb.SubmitTransactionRequest
}

func SendCommand(w io.Writer, f *SendCommandFlags, rf *RootFlags, req *SendCommandRequest) error {
	p := printer.NewInteractivePrinter(w)

	if rf.Output == flags.InteractiveOutput && version.IsUnreleased() {
		str := p.String()
		str.CrossMark().DangerText("You are running an unreleased version of the Vega wallet (").DangerText(coreversion.Get()).DangerText(").").NextLine()
		str.Pad().DangerText("Use it at your own risk!").NextSection()
		p.Print(str)
	}

	log, err := BuildLogger(rf.Output, req.LogLevel)
	if err != nil {
		return err
	}
	defer vglog.Sync(log)

	// Login early to check passphrase before running any query
	store, err := wallets.InitialiseStore(rf.Home)
	if err != nil {
		return fmt.Errorf("couldn't initialise wallets store: %w", err)
	}
	handler := wallets.NewHandler(store)
	err = handler.LoginWallet(req.Wallet, req.Passphrase)
	if err != nil {
		return fmt.Errorf("couldn't login to the wallet %s: %w", req.Wallet, err)
	}
	defer handler.LogoutWallet(req.Wallet)

	var hosts []string
	if len(req.Network) != 0 {
		hosts, err = getHostsFromNetwork(rf, req.Network)
		if err != nil {
			return err
		}
	} else {
		hosts = []string{req.NodeAddress}
	}

	if !f.NoVersionCheck {
		networkVersion, err := walletversion.GetNetworkVersionThroughGRPC(hosts)
		if err != nil {
			return err
		}
		if networkVersion != coreversion.Get() {
			return fmt.Errorf("software is not compatible with this network: network is running version %s but wallet software has version %s", networkVersion, coreversion.Get())
		}
	}

	forwarder, err := node.NewForwarder(log.Named("forwarder"), network.GRPCConfig{
		Hosts:   hosts,
		Retries: req.Retries,
	})

	log = log.Named("command")

	if err != nil {
		return fmt.Errorf("couldn't initialise the node forwarder: %w", err)
	}
	defer func() {
		if err = forwarder.Stop(); err != nil {
			log.Warn("couldn't stop the forwarder", zap.Error(err))
		}
	}()

	if rf.Output == flags.InteractiveOutput {
		p.Print(p.String().BlueArrow().InfoText("Logs").NextLine())
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), ForwarderRequestTimeout)
	defer cancelFn()

	log.Info("retrieving block height")
	blockData, cltIdx, err := forwarder.LastBlockHeightAndHash(ctx)
	if err != nil {
		return fmt.Errorf("couldn't get last block height: %w", err)
	}

	if blockData.ChainId == "" {
		return fmt.Errorf("network does not have a chainid")
	}

	log.Info(fmt.Sprintf("last block height found: %d", blockData.Height))

	tx, err := handler.SignTx(req.Wallet, req.Request, blockData.Height, blockData.ChainId)
	if err != nil {
		log.Error("couldn't sign transaction", zap.Error(err))
		return fmt.Errorf("couldn't sign transaction: %w", err)
	}

	log.Info("transaction successfully signed", zap.String("signature", tx.Signature.Value))

	// generate a random transaction hash
	tid := crypto.RandomHash()

	// generate proof of work for the block hash, transaction id and given difficulty required and supported hash function
	powNonce, _, err := crypto.PoW(blockData.Hash, tid, uint(blockData.SpamPowDifficulty), blockData.SpamPowHashFunction)
	if err != nil {
		return fmt.Errorf("couldn't generate proof of work for last block height: %w", err)
	}
	tx.Pow = &commandspb.ProofOfWork{
		Tid:   tid,
		Nonce: powNonce,
	}

	log.Info("calculated proof of work for transaction", zap.String("signature", tx.Signature.Value))

	resp, err := forwarder.SendTx(ctx, tx, api.SubmitTransactionRequest_TYPE_ASYNC, cltIdx)
	if err != nil {
		log.Error("couldn't send transaction", zap.Error(err))
		return fmt.Errorf("couldn't send transaction: %w", err)
	}

	if !resp.Success {
		log.Error("transaction failed", zap.String("err", resp.Data), zap.Uint32("code", resp.Code))
		return fmt.Errorf("transaction failed: %s", resp.Data)
	}

	log.Info("transaction successfully sent", zap.String("hash", resp.TxHash))

	return nil
}
