package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	vgterm "code.vegaprotocol.io/shared/libs/term"
	vglog "code.vegaprotocol.io/shared/libs/zap"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/wallet/network"
	netstore "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"code.vegaprotocol.io/vega/wallet/node"
	"code.vegaprotocol.io/vega/wallet/proxy"
	"code.vegaprotocol.io/vega/wallet/service"
	svcstore "code.vegaprotocol.io/vega/wallet/service/store/v1"
	"code.vegaprotocol.io/vega/wallet/wallets"
	"github.com/golang/protobuf/jsonpb"

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const MaxConsentRequests = 100

var ErrEnableAutomaticConsentFlagIsRequiredWithoutTTY = errors.New("--automatic-consent flag is required without TTY")

var (
	ErrProgramIsNotInitialised = errors.New("first, you need initialise the program, using the `init` command")

	runServiceLong = cli.LongDesc(`
		Start a Vega wallet service behind an HTTP server.

		By default, every incoming transactions will have to be reviewed in the
		terminal.

		To terminate the service, hit ctrl+c.

		NOTE: The --output flag is ignored in this command.
	`)

	runServiceExample = cli.Examples(`
		# Start the service
		{{.Software}} service run --network NETWORK

		# Start the service and open the console in the default browser
		{{.Software}} service run --network NETWORK --with-console

		# Start the service without opening the console
		{{.Software}} service run --network NETWORK --with-console --no-browser

		# Start the service and open the token dApp in the default browser
		{{.Software}} service run --network NETWORK --with-token-dapp

		# Start the service without opening the token dApp
		{{.Software}} service run --network NETWORK --with-token-dapp --no-browser

		# Start the service with automatic consent of incoming transactions
		{{.Software}} service run --network NETWORK --automatic-consent
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
	cmd.Flags().BoolVar(&f.WithConsole,
		"with-console",
		false,
		"Start the Vega console behind a proxy and open it in the default browser",
	)
	cmd.Flags().BoolVar(&f.WithTokenDApp,
		"with-token-dapp",
		false,
		"Start the Vega Token dApp behind a proxy and open it in the default browser",
	)
	cmd.Flags().BoolVar(&f.NoBrowser,
		"no-browser",
		false,
		"Do not open the default browser when starting applications",
	)
	cmd.Flags().BoolVar(&f.EnableAutomaticConsent,
		"automatic-consent",
		false,
		"Automatically approve incoming transaction. Only use this flag when you have absolute trust in incoming transactions! No logs on standard output.",
	)

	autoCompleteNetwork(cmd, rf.Home)

	return cmd
}

type RunServiceFlags struct {
	Network                string
	WithConsole            bool
	WithTokenDApp          bool
	NoBrowser              bool
	EnableAutomaticConsent bool
}

func (f *RunServiceFlags) Validate() error {
	if len(f.Network) == 0 {
		return flags.FlagMustBeSpecifiedError("network")
	}

	if f.NoBrowser && !f.WithConsole && !f.WithTokenDApp {
		return flags.OneOfParentsFlagMustBeSpecifiedError("no-browser", "with-console", "with-token-dapp")
	}

	return nil
}

func RunService(w io.Writer, rf *RootFlags, f *RunServiceFlags) error {
	store, err := wallets.InitialiseStore(rf.Home)
	if err != nil {
		return fmt.Errorf("couldn't initialise wallets store: %w", err)
	}

	handler := wallets.NewHandler(store)

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

	if err := verifyNetworkConfig(cfg, f); err != nil {
		return err
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

	p := printer.NewInteractivePrinter(w)

	p.Print(p.String().CheckMark().Text("Service logs located at: ").SuccessText(svcLogPath).NextLine())
	p.Print(p.String().CheckMark().Text("CLI logs located at: ").SuccessText(cliLogPath).NextLine())

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

	srv, err := service.NewService(svcLog.Named("api"), cfg, handler, auth, forwarder, policy)
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

	var cs *proxy.Proxy
	if f.WithConsole {
		cs = startConsole(cliLog, f, cfg, cancel, p)
		defer func() {
			if err = cs.Stop(); err != nil {
				cliLog.Error("failed to stop console proxy", zap.Error(err))
				p.Print(p.String().DangerBangMark().Text("Failed to stop console proxy: ").DangerText(err.Error()).NextLine())
				return
			}
			cliLog.Info("Console proxy stopped with success")
			p.Print(p.String().CheckMark().Text("Console proxy stopped").NextLine())
		}()
	}

	var tokenDApp *proxy.Proxy
	if f.WithTokenDApp {
		tokenDApp = startTokenDApp(cliLog, f, cfg, cancel, p)
		defer func() {
			if err = tokenDApp.Stop(); err != nil {
				cliLog.Error("failed to stop token dApp proxy", zap.Error(err))
				p.Print(p.String().DangerBangMark().Text("Failed to stop token dApp proxy: ").DangerText(err.Error()).NextLine())
				return
			}
			cliLog.Info("Token dApp proxy stopped with success")
			p.Print(p.String().CheckMark().Text("Token dApp proxy stopped").NextLine())
		}()
	}

	waitSig(ctx, cancel, cliLog, consentRequests, sentTransactions, p)

	return nil
}

func verifyNetworkConfig(cfg *network.Network, f *RunServiceFlags) error {
	if err := cfg.EnsureCanConnectGRPCNode(); err != nil {
		return err
	}
	if f.WithConsole {
		if err := cfg.EnsureCanConnectConsole(); err != nil {
			return err
		}
	}
	if f.WithTokenDApp {
		if err := cfg.EnsureCanConnectTokenDApp(); err != nil {
			return err
		}
	}
	return nil
}

func startConsole(log *zap.Logger, f *RunServiceFlags, cfg *network.Network, cancel context.CancelFunc, p *printer.InteractivePrinter) *proxy.Proxy {
	cs := proxy.NewProxy(cfg.Console.LocalPort, cfg.Console.URL, cfg.API.GRPC.Hosts[0])
	consoleLocalProxyURL := cs.GetLocalProxyURL()
	go func() {
		defer cancel()
		p.Print(p.String().CheckMark().Text("Starting console proxy for ").Bold(cfg.Console.URL).Text(" at: ").SuccessText(consoleLocalProxyURL).NextLine())
		log.Info("starting console proxy", zap.String("target-url", cfg.Console.URL), zap.String("proxy-url", consoleLocalProxyURL))
		if err := cs.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Failed to start the console proxy", zap.Error(err))
			p.Print(p.String().DangerBangMark().Text("Failed to start console proxy: ").DangerText(consoleLocalProxyURL).NextLine())
		}
	}()

	if !f.NoBrowser {
		p.Print(p.String().CheckMark().Text("Opening browser for console proxy at: ").SuccessText(consoleLocalProxyURL).NextLine())
		log.Info("opening browser for console proxy", zap.String("url", consoleLocalProxyURL))
		if err := open.Run(consoleLocalProxyURL); err != nil {
			log.Error("unable to open the application in the default browser", zap.Error(err))
			p.Print(p.String().DangerBangMark().Text("Failed to open the browser for console proxy: ").DangerText(err.Error()).NextLine())
		}
	}

	return cs
}

func startTokenDApp(log *zap.Logger, f *RunServiceFlags, cfg *network.Network, cancel context.CancelFunc, p *printer.InteractivePrinter) *proxy.Proxy {
	tokenDApp := proxy.NewProxy(cfg.TokenDApp.LocalPort, cfg.TokenDApp.URL, cfg.API.GRPC.Hosts[0])
	tokenDAppLocalProxyURL := tokenDApp.GetLocalProxyURL()
	go func() {
		defer cancel()
		p.Print(p.String().CheckMark().Text("Starting token dApp proxy for ").Bold(cfg.TokenDApp.URL).Text(" at: ").SuccessText(tokenDAppLocalProxyURL).NextLine())
		log.Info("starting token dApp proxy", zap.String("target-url", cfg.TokenDApp.URL), zap.String("proxy-url", tokenDAppLocalProxyURL))

		if err := tokenDApp.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start the token dApp proxy", zap.Error(err))
			p.Print(p.String().DangerBangMark().Text("Failed to start token dApp proxy: ").DangerText(tokenDAppLocalProxyURL).NextLine())
		}
	}()

	if !f.NoBrowser {
		p.Print(p.String().CheckMark().Text("Opening browser for token dApp proxy at: ").SuccessText(tokenDAppLocalProxyURL).NextLine())
		log.Info("opening browser for token dApp proxy", zap.String("url", tokenDAppLocalProxyURL))
		if err := open.Run(tokenDAppLocalProxyURL); err != nil {
			log.Error("unable to open the token dApp in the default browser", zap.Error(err))
			p.Print(p.String().DangerBangMark().Text("Failed to open the browser for token dApp proxy: ").DangerText(err.Error()).NextLine())
		}
	}
	return tokenDApp
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(
	ctx context.Context,
	cancelFunc context.CancelFunc,
	log *zap.Logger,
	consentRequests chan service.ConsentRequest,
	sentTransactions chan service.SentTransaction,
	p *printer.InteractivePrinter,
) {
	gracefulStop := make(chan os.Signal, 1)
	defer close(gracefulStop)

	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	signal.Notify(gracefulStop, syscall.SIGQUIT)

	go func() {
		if err := handleConsentRequests(ctx, log, consentRequests, sentTransactions, p); err != nil {
			cancelFunc()
		}
	}()

	for {
		select {
		case sig := <-gracefulStop:
			log.Info("caught signal", zap.String("signal", fmt.Sprintf("%+v", sig)))
			cancelFunc()
			return
		case <-ctx.Done():
			// nothing to do
			return
		}
	}
}

func handleConsentRequests(ctx context.Context, log *zap.Logger, consentRequests chan service.ConsentRequest, sentTransactions chan service.SentTransaction, p *printer.InteractivePrinter) error {
	for {
		select {
		case <-ctx.Done():
			// nothing to do
			return nil
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
