package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	vgterm "code.vegaprotocol.io/vega/libs/term"
	vglog "code.vegaprotocol.io/vega/libs/zap"
	"code.vegaprotocol.io/vega/paths"
	coreversion "code.vegaprotocol.io/vega/version"
	walletapi "code.vegaprotocol.io/vega/wallet/api"
	walletnode "code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/api/pipeline"
	"code.vegaprotocol.io/vega/wallet/network"
	netstore "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"code.vegaprotocol.io/vega/wallet/node"
	"code.vegaprotocol.io/vega/wallet/service"
	svcstore "code.vegaprotocol.io/vega/wallet/service/store/v1"
	"code.vegaprotocol.io/vega/wallet/version"
	"code.vegaprotocol.io/vega/wallet/wallets"
	"github.com/golang/protobuf/jsonpb"
	"golang.org/x/term"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const MaxConsentRequests = 100

var (
	ErrEnableAutomaticConsentFlagIsRequiredWithoutTTY = errors.New("--automatic-consent flag is required without TTY")
	ErrMsysUnsupported                                = errors.New("this command is not supported on msys, please use a standard windows terminal")
	ErrNoHostSpecified                                = errors.New("no host specified in the configuration")
)

var (
	ErrProgramIsNotInitialised = errors.New("first, you need initialise the program, using the `init` command")

	runServiceLong = cli.LongDesc(`
		Start a Vega wallet service behind an HTTP server.

		By default, every incoming transactions will have to be reviewed in the
		terminal.

		To terminate the service, hit ctrl+c.

		NOTE: The --output flag is ignored in this command.

		WARNING: This command is not supported on msys, due to some system
        incompatibilities with the user input management.
		Non-exhaustive list of affected systems: Cygwin, minty, git-bash.
	`)

	runServiceExample = cli.Examples(`
		# Start the service
		{{.Software}} service run --network NETWORK

		# Start the service with automatic consent of incoming transactions for API v1.
		{{.Software}} service run --network NETWORK --automatic-consent

		# Start the service without verifying network version compatibility
		{{.Software}} service run --network NETWORK --no-version-check
	`)
)

type RunServiceHandler func(io.Writer, *RootFlags, *RunServiceFlags) error

func NewCmdRunService(w io.Writer, rf *RootFlags) *cobra.Command {
	return BuildCmdRunService(w, RunService, rf)
}

func BuildCmdRunService(w io.Writer, handler RunServiceHandler, rf *RootFlags) *cobra.Command {
	f := &RunServiceFlags{}

	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Start the Vega wallet service",
		Long:    runServiceLong,
		Example: runServiceExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := f.Validate(); err != nil {
				return err
			}

			if err := handler(w, rf, f); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Network,
		"network", "n",
		"",
		"Network configuration to use",
	)
	cmd.Flags().BoolVar(&f.EnableAutomaticConsent,
		"automatic-consent",
		false,
		"Automatically approve incoming transaction. Only use this flag when you have absolute trust in incoming transactions! No logs on standard output. Only works for API v1.",
	)
	cmd.PersistentFlags().BoolVar(&f.NoVersionCheck,
		"no-version-check",
		false,
		"Do not check for new version of the Vega wallet",
	)

	autoCompleteNetwork(cmd, rf.Home)

	return cmd
}

type RunServiceFlags struct {
	Network                string
	EnableAutomaticConsent bool
	NoVersionCheck         bool
}

func (f *RunServiceFlags) Validate() error {
	if len(f.Network) == 0 {
		return flags.FlagMustBeSpecifiedError("network")
	}

	return nil
}

func RunService(w io.Writer, rf *RootFlags, f *RunServiceFlags) error {
	if err := ensureNotRunningInMsys(); err != nil {
		return err
	}

	p := printer.NewInteractivePrinter(w)

	if rf.Output == flags.InteractiveOutput && version.IsUnreleased() {
		str := p.String()
		str.CrossMark().DangerText("You are running an unreleased version of the Vega wallet (").DangerText(coreversion.Get()).DangerText(").").NextLine()
		str.Pad().DangerText("Use it at your own risk!").NextSection()
		p.Print(str)
	}

	walletStore, err := wallets.InitialiseStore(rf.Home)
	if err != nil {
		return fmt.Errorf("couldn't initialise wallets store: %w", err)
	}

	handler := wallets.NewHandler(walletStore)

	vegaPaths := paths.New(rf.Home)
	netStore, err := netstore.InitialiseStore(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise network store: %w", err)
	}

	exists, err := netStore.NetworkExists(f.Network)
	if err != nil {
		return fmt.Errorf("couldn't verify the network existence: %w", err)
	}
	if !exists {
		return network.NewNetworkDoesNotExistError(f.Network)
	}

	cfg, err := netStore.GetNetwork(f.Network)
	if err != nil {
		return fmt.Errorf("couldn't initialise network store: %w", err)
	}

	if err := cfg.EnsureCanConnectGRPCNode(); err != nil {
		return err
	}

	if !f.NoVersionCheck {
		networkVersion, err := getNetworkVersion(cfg.API.GRPC.Hosts[0])
		if err != nil {
			return err
		}
		if networkVersion != coreversion.Get() {
			return fmt.Errorf("software is not compatible with this network: network is running version %s but wallet software has version %s", networkVersion, coreversion.Get())
		}
	}

	svcLog, svcLogPath, err := BuildJSONLogger(cfg.Level.String(), vegaPaths, paths.WalletServiceLogsHome)
	if err != nil {
		return err
	}
	defer vglog.Sync(svcLog)

	svcLog = svcLog.Named("service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svcStore, err := svcstore.InitialiseStore(paths.New(rf.Home))
	if err != nil {
		return fmt.Errorf("couldn't initialise service store: %w", err)
	}

	if isInit, err := service.IsInitialised(svcStore); err != nil {
		return fmt.Errorf("couldn't verify service initialisation state: %w", err)
	} else if !isInit {
		return ErrProgramIsNotInitialised
	}

	auth, err := service.NewAuth(svcLog.Named("auth"), svcStore, cfg.TokenExpiry.Get())
	if err != nil {
		return fmt.Errorf("couldn't initialise authentication: %w", err)
	}

	forwarder, err := node.NewForwarder(svcLog.Named("forwarder"), cfg.API.GRPC)
	if err != nil {
		return fmt.Errorf("couldn't initialise the node forwarder: %w", err)
	}

	cliLog, cliLogPath, err := BuildJSONLogger(cfg.Level.String(), vegaPaths, paths.WalletCLILogsHome)
	if err != nil {
		return err
	}
	defer vglog.Sync(cliLog)

	cliLog = cliLog.Named("command")

	str := p.String()
	str.CheckMark().Text("Service logs located at: ").SuccessText(svcLogPath).NextLine()
	str.CheckMark().Text("CLI logs located at: ").SuccessText(cliLogPath).NextLine()
	p.Print(str)

	consentRequests := make(chan service.ConsentRequest, MaxConsentRequests)
	defer close(consentRequests)
	sentTransactions := make(chan service.SentTransaction)
	defer close(sentTransactions)

	var policy service.Policy
	if vgterm.HasTTY() {
		cliLog.Info("TTY detected")
		if f.EnableAutomaticConsent {
			cliLog.Info("Automatic consent enabled")
			p.Print(p.String().WarningBangMark().WarningText("Automatic consent enabled").NextLine())
			policy = service.NewAutomaticConsentPolicy()
		} else {
			cliLog.Info("Explicit consent enabled")
			p.Print(p.String().CheckMark().Text("Explicit consent enabled").NextLine())
			policy = service.NewExplicitConsentPolicy(ctx, consentRequests, sentTransactions)
		}
	} else {
		cliLog.Info("No TTY detected")
		if !f.EnableAutomaticConsent {
			cliLog.Error("Explicit consent can't be used when no TTY is attached to the process")
			return ErrEnableAutomaticConsentFlagIsRequiredWithoutTTY
		}
		cliLog.Info("Automatic consent enabled")
		policy = service.NewAutomaticConsentPolicy()
	}

	receptionChan := make(chan pipeline.Envelope, 100)
	responseChan := make(chan pipeline.Envelope, 100)
	seqPipeline := pipeline.NewSequentialPipeline(ctx, receptionChan, responseChan)
	jsonrpcLog := svcLog.Named("json-rpc")
	nodeSelector, err := buildNodeSelector(jsonrpcLog, cfg.API.GRPC)
	if err != nil {
		cliLog.Error("Couldn't instantiate node API", zap.Error(err))
		return fmt.Errorf("couldn't instantiate node API: %w", err)
	}
	apiV2, err := walletapi.RestrictedAPI(jsonrpcLog, walletStore, seqPipeline, nodeSelector)
	if err != nil {
		return fmt.Errorf("couldn't instantiate JSON-RPC API: %w", err)
	}

	srv, err := service.NewService(svcLog.Named("api"), cfg, apiV2, handler, auth, forwarder, policy)
	if err != nil {
		return err
	}

	go func() {
		defer cancel()
		serviceHost := fmt.Sprintf("http://%v:%v", cfg.Host, cfg.Port)
		p.Print(p.String().CheckMark().Text("Starting HTTP service at: ").SuccessText(serviceHost).NextLine())
		cliLog.Info("starting HTTP service", zap.String("url", serviceHost))
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			cliLog.Error("failed to start HTTP server", zap.Error(err))
			p.Print(p.String().DangerBangMark().Text("Failed to start HTTP server: ").DangerText(err.Error()).NextLine())
		}
	}()

	defer func() {
		if err = srv.Stop(); err != nil {
			cliLog.Error("Failed to stop HTTP server", zap.Error(err))
			p.Print(p.String().DangerBangMark().Text("HTTP service stopped with error: ").DangerText(err.Error()).NextLine())
			return
		}
		cliLog.Info("HTTP server stopped with success")
		p.Print(p.String().CheckMark().Text("HTTP service stopped").NextLine())
	}()

	waitSig(ctx, cancel, cliLog, consentRequests, sentTransactions, receptionChan, responseChan, p)

	return nil
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(
	ctx context.Context,
	cancelFunc context.CancelFunc,
	log *zap.Logger,
	consentRequests chan service.ConsentRequest,
	sentTransactions chan service.SentTransaction,
	receptionChan <-chan pipeline.Envelope,
	responseChan chan<- pipeline.Envelope,
	p *printer.InteractivePrinter,
) {
	gracefulStop := make(chan os.Signal, 1)
	defer close(gracefulStop)

	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	signal.Notify(gracefulStop, syscall.SIGQUIT)

	go func() {
		if err := handleConsentRequests(ctx, log, consentRequests, sentTransactions, receptionChan, responseChan, p); err != nil {
			cancelFunc()
		}
	}()

	for {
		select {
		case sig := <-gracefulStop:
			log.Info("caught signal", zap.String("signal", fmt.Sprintf("%+v", sig)))
			p.Print(p.String().NextSection())
			cancelFunc()
			return
		case <-ctx.Done():
			// nothing to do
			return
		}
	}
}

func handleConsentRequests(ctx context.Context, log *zap.Logger, consentRequests chan service.ConsentRequest, sentTransactions chan service.SentTransaction, receptionChan <-chan pipeline.Envelope, responseChan chan<- pipeline.Envelope, p *printer.InteractivePrinter) error {
	p.Print(p.String().NextSection())
	for {
		select {
		case <-ctx.Done():
			// nothing to do
			return nil
		case envelop := <-receptionChan:
			if err := handleEnvelop(envelop, responseChan, p); err != nil {
				return err
			}
		case consentRequest := <-consentRequests:
			m := jsonpb.Marshaler{Indent: "    "}
			marshalledTx, err := m.MarshalToString(consentRequest.Tx)
			if err != nil {
				log.Error("couldn't marshal transaction from consent request", zap.Error(err))
				return err
			}

			str := p.String()
			str.BlueArrow().Text("New transaction received: ").NextLine()
			str.InfoText(marshalledTx).NextLine()
			p.Print(str)

			if flags.DoYouApproveTx() {
				log.Info("user approved the signing of the transaction", zap.Any("transaction", marshalledTx))
				consentRequest.Confirmation <- service.ConsentConfirmation{Decision: true}
				p.Print(p.String().CheckMark().SuccessText("Transaction approved").NextLine())

				sentTx := <-sentTransactions
				log.Info("transaction sent", zap.Any("ID", sentTx.TxID), zap.Any("hash", sentTx.TxHash))
				if sentTx.Error != nil {
					log.Error("transaction failed", zap.Any("transaction", marshalledTx))
					p.Print(p.String().DangerBangMark().DangerText("Transaction failed").NextLine())
					p.Print(p.String().DangerBangMark().DangerText("Error: ").DangerText(sentTx.Error.Error()).NextSection())
				} else {
					log.Info("transaction sent", zap.Any("hash", sentTx.TxHash))
					p.Print(p.String().CheckMark().Text("Transaction with hash ").SuccessText(sentTx.TxHash).Text(" sent!").NextSection())
				}
			} else {
				log.Info("user rejected the signing of the transaction", zap.Any("transaction", marshalledTx))
				consentRequest.Confirmation <- service.ConsentConfirmation{Decision: false}
				p.Print(p.String().DangerBangMark().DangerText("Transaction rejected").NextSection())
			}
		}
	}
}

func handleEnvelop(envelop pipeline.Envelope, responseChan chan<- pipeline.Envelope, p *printer.InteractivePrinter) error {
	switch content := envelop.Content.(type) {
	case pipeline.RequestWalletConnectionReview:
		p.Print(p.String().BlueArrow().Text("The application \"").InfoText(content.Hostname).Text("\" wants to connect to your wallet.").NextLine())
		approved := yesOrNo(p.String().QuestionMark().Text("Do you approve connecting your wallet to this application?"), p)
		if approved {
			p.Print(p.String().CheckMark().Text("Connection approved.").NextLine())
		} else {
			p.Print(p.String().CrossMark().Text("Connection rejected.").NextSection())
		}
		responseChan <- pipeline.Envelope{
			TraceID: envelop.TraceID,
			Content: pipeline.Decision{
				Approved: approved,
			},
		}
	case pipeline.RequestWalletSelection:
		str := p.String().BlueArrow().Text("Here are the available wallets:").NextLine()
		for _, w := range content.AvailableWallets {
			str.ListItem().Text("- ").InfoText(w).NextLine()
		}
		p.Print(str)
		selectedWallet := readInput(p.String().QuestionMark().Text("Which wallet do you want to use? "), p, content.AvailableWallets)
		passphrase := readPassphrase(p.String().BlueArrow().Text("Enter the passphrase for the wallet \"").InfoText(selectedWallet).Text("\": "), p)
		responseChan <- pipeline.Envelope{
			TraceID: envelop.TraceID,
			Content: walletapi.SelectedWallet{
				Wallet:     selectedWallet,
				Passphrase: passphrase,
			},
		}
	case pipeline.RequestPassphrase:
		passphrase := readPassphrase(p.String().BlueArrow().Text("Enter the passphrase for the wallet \"").InfoText(content.Wallet).Text("\": "), p)
		responseChan <- pipeline.Envelope{
			TraceID: envelop.TraceID,
			Content: pipeline.EnteredPassphrase{
				Passphrase: passphrase,
			},
		}
	case pipeline.ErrorOccurred:
		if content.Type == string(walletapi.InternalError) {
			str := p.String().DangerBangMark().DangerText("An internal error occurred: ").DangerText(content.Error).NextLine()
			str.DangerBangMark().DangerText("The request has been canceled.").NextSection()
			p.Print(str)
		} else if content.Type == string(walletapi.ClientError) {
			p.Print(p.String().DangerBangMark().DangerText(content.Error).NextLine())
		} else {
			p.Print(p.String().DangerBangMark().DangerText(fmt.Sprintf("Error: %s (%s)", content.Error, content.Type)).NextLine())
		}
	case pipeline.RequestSucceeded:
		p.Print(p.String().CheckMark().SuccessText("Request succeeded").NextSection())
	case pipeline.RequestPermissionsReview:
		str := p.String().BlueArrow().Text("The application \"").InfoText(content.Hostname).Text("\" want to update the permissions for the wallet \"").InfoText(content.Wallet).Text("\":").NextLine()
		for perm, access := range content.Permissions {
			str.ListItem().Text("- ").InfoText(perm).Text(": ").InfoText(access).NextLine()
		}
		p.Print(str)
		approved := yesOrNo(p.String().QuestionMark().Text("Do you approve this update?"), p)
		if approved {
			p.Print(p.String().CheckMark().Text("Permissions update approved.").NextLine())
		} else {
			p.Print(p.String().CrossMark().Text("Permissions update rejected.").NextSection())
		}
		responseChan <- pipeline.Envelope{
			TraceID: envelop.TraceID,
			Content: pipeline.Decision{
				Approved: approved,
			},
		}
	case pipeline.RequestTransactionReview:
		str := p.String().BlueArrow().Text("The application \"").InfoText(content.Hostname).Text("\" wants to send the following transaction:").NextLine()
		str.Pad().Text("Using the key: ").InfoText(content.PublicKey).NextLine()
		str.Pad().Text("From the wallet: ").InfoText(content.Wallet).NextLine()
		fmtCmd := strings.Replace("  "+content.Transaction, "\n", "\n  ", -1)
		str.InfoText(fmtCmd).NextLine()
		p.Print(str)
		approved := yesOrNo(p.String().QuestionMark().Text("Do you approve this transaction?"), p)
		if approved {
			p.Print(p.String().CheckMark().Text("Transaction approved.").NextLine())
		} else {
			p.Print(p.String().CrossMark().Text("Command rejected.").NextSection())
		}
		responseChan <- pipeline.Envelope{
			TraceID: envelop.TraceID,
			Content: pipeline.Decision{
				Approved: approved,
			},
		}
	case pipeline.TransactionStatus:
		str := p.String()
		if content.Error != nil {
			str.DangerBangMark().DangerText("The transaction failed.").NextLine()
			str.Pad().DangerText(content.Error.Error()).NextLine()
			str.Pad().Text("Sent at: ").Text(content.SentAt.Format(time.ANSIC)).NextSection()
		} else {
			str.CheckMark().SuccessText("The transaction has been delivered").NextLine()
			str.Pad().Text("Transaction hash: ").SuccessText(content.TxHash).NextLine()
			str.Pad().Text("Sent at: ").Text(content.SentAt.Format(time.ANSIC)).NextSection()
		}
		p.Print(str)
	default:
		panic(fmt.Sprintf("unhandled pipeline envelop: %v", content))
	}
	return nil
}

func readInput(question *printer.FormattedString, p *printer.InteractivePrinter, options []string) string {
	reader := bufio.NewReader(os.Stdin)
	for {
		p.Print(question)
		answer, err := reader.ReadString('\n')
		if err != nil {
			panic(fmt.Errorf("couldn't read input: %w", err))
		}

		answer = strings.ToLower(strings.Trim(answer, " \r\n\t"))

		if len(options) == 0 {
			return answer
		}
		for _, option := range options {
			if answer == option {
				return answer
			}
		}
		p.Print(p.String().DangerBangMark().DangerText(fmt.Sprintf("%q is not a valid option", answer)).NextSection())
	}
}

func yesOrNo(question *printer.FormattedString, p *printer.InteractivePrinter) bool {
	question.Text(" (yes/no) ")
	reader := bufio.NewReader(os.Stdin)
	for {
		p.Print(question)
		answer, err := reader.ReadString('\n')
		if err != nil {
			panic(fmt.Errorf("couldn't read input: %w", err))
		}

		answer = strings.ToLower(strings.Trim(answer, " \r\n\t"))

		switch answer {
		case "yes", "y":
			return true
		case "no", "n":
			return false
		default:
			p.Print(p.String().DangerBangMark().DangerText(fmt.Sprintf("%q is not a valid answer, enter \"yes\" or \"no\"\n", answer)))
		}
	}
}

// ensureNotRunningInMsys verifies if the underlying shell is not running on
// msys.
// This command is not supported on msys, due to some system incompatibilities
// with the user input management.
// Non-exhaustive list of affected systems: Cygwin, minty, git-bash.
func ensureNotRunningInMsys() error {
	ms := os.Getenv("MSYSTEM")
	if ms != "" {
		return ErrMsysUnsupported
	}
	return nil
}

func readPassphrase(question *printer.FormattedString, p *printer.InteractivePrinter) string {
	p.Print(question)
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		panic(fmt.Errorf("couldn't read passphrase: %w", err))
	}
	p.Print(p.String().NextLine())
	return string(passphrase)
}

func buildNodeSelector(log *zap.Logger, nodesConfig network.GRPCConfig) (walletapi.NodeSelector, error) {
	if len(nodesConfig.Hosts) == 0 {
		return nil, ErrNoHostSpecified
	}

	nodes := make([]walletapi.Node, 0, len(nodesConfig.Hosts))
	for _, host := range nodesConfig.Hosts {
		n, err := walletnode.NewGRPCNode(log.Named("grpc-node"), host, nodesConfig.Retries)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialize node for %q: %w", host, err)
		}
		nodes = append(nodes, n)
	}

	nodeSelector, err := walletnode.NewRoundRobinSelector(log.Named("round-robin-selector"), nodes...)
	if err != nil {
		return nil, fmt.Errorf("couldn't instantiate round-robin node selector: %w", err)
	}

	return nodeSelector, nil
}
