package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	vgclose "code.vegaprotocol.io/vega/libs/close"
	vgjob "code.vegaprotocol.io/vega/libs/job"
	vgterm "code.vegaprotocol.io/vega/libs/term"
	vgzap "code.vegaprotocol.io/vega/libs/zap"
	"code.vegaprotocol.io/vega/paths"
	coreversion "code.vegaprotocol.io/vega/version"
	walletapi "code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/interactor"
	netStoreV1 "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"code.vegaprotocol.io/vega/wallet/preferences"
	"code.vegaprotocol.io/vega/wallet/service"
	svcStoreV1 "code.vegaprotocol.io/vega/wallet/service/store/v1"
	serviceV1 "code.vegaprotocol.io/vega/wallet/service/v1"
	serviceV2 "code.vegaprotocol.io/vega/wallet/service/v2"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections/store/longliving/v1"
	sessionStoreV1 "code.vegaprotocol.io/vega/wallet/service/v2/connections/store/session/v1"
	"code.vegaprotocol.io/vega/wallet/version"
	"code.vegaprotocol.io/vega/wallet/wallets"
	"github.com/golang/protobuf/jsonpb"
	"github.com/muesli/cancelreader"
	"golang.org/x/term"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const MaxConsentRequests = 100

var (
	ErrEnableAutomaticConsentFlagIsRequiredWithoutTTY = errors.New("--automatic-consent flag is required without TTY")
	ErrMsysUnsupported                                = errors.New("this command is not supported on msys, please use a standard windows terminal")
)

var (
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

		# Start the service with automatic consent of incoming transactions
		{{.Software}} service run --network NETWORK --automatic-consent

		# Start the service without verifying network version compatibility
		{{.Software}} service run --network NETWORK --no-version-check
	`)
)

type ServicePreCheck func(rf *RootFlags) error

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
			if err := f.Validate(rf); err != nil {
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
		"Automatically approve incoming transaction. Only use this flag when you have absolute trust in incoming transactions!",
	)
	cmd.Flags().BoolVar(&f.LoadTokens,
		"load-tokens",
		false,
		"Load the sessions with long-living tokens",
	)
	cmd.PersistentFlags().BoolVar(&f.NoVersionCheck,
		"no-version-check",
		false,
		"Do not check the network version compatibility",
	)
	cmd.Flags().StringVar(&f.TokensPassphraseFile,
		"tokens-passphrase-file",
		"",
		"Path to the file containing the tokens database passphrase",
	)

	autoCompleteNetwork(cmd, rf.Home)

	return cmd
}

type RunServiceFlags struct {
	Network                string
	EnableAutomaticConsent bool
	LoadTokens             bool
	TokensPassphraseFile   string
	NoVersionCheck         bool
	tokensPassphrase       string
}

func (f *RunServiceFlags) Validate(rf *RootFlags) error {
	if len(f.Network) == 0 {
		return flags.MustBeSpecifiedError("network")
	}

	if !f.LoadTokens && f.TokensPassphraseFile != "" {
		return flags.OneOfParentsFlagMustBeSpecifiedError("tokens-passphrase-file", "load-tokens")
	}

	if f.LoadTokens {
		if err := ensureAPITokenStoreIsInit(rf); err != nil {
			return err
		}
		passphrase, err := flags.GetPassphraseWithOptions(flags.PassphraseOptions{Name: "tokens"}, f.TokensPassphraseFile)
		if err != nil {
			return err
		}
		f.tokensPassphrase = passphrase
	}

	return nil
}

func RunService(w io.Writer, rf *RootFlags, f *RunServiceFlags) error {
	if err := ensureNotRunningInMsys(); err != nil {
		return err
	}

	p := printer.NewInteractivePrinter(w)

	vegaPaths := paths.New(rf.Home)

	cliLog, cliLogPath, _, err := buildJSONFileLogger(vegaPaths, paths.WalletCLILogsHome, "info")
	if err != nil {
		return err
	}
	defer vgzap.Sync(cliLog)
	cliLog = cliLog.Named("command")

	if rf.Output == flags.InteractiveOutput && version.IsUnreleased() {
		cliLog.Warn("Current software is an unreleased version", zap.String("version", coreversion.Get()))
		str := p.String()
		str.CrossMark().DangerText("You are running an unreleased version of the Vega wallet (").DangerText(coreversion.Get()).DangerText(").").NextLine()
		str.Pad().DangerText("Use it at your own risk!").NextSection()
		p.Print(str)
	} else {
		cliLog.Warn("Current software is a released version", zap.String("version", coreversion.Get()))
	}

	p.Print(p.String().CheckMark().Text("CLI logs located at: ").SuccessText(cliLogPath).NextLine())

	closer := vgclose.NewCloser()
	defer closer.CloseAll()

	walletStore, err := wallets.InitialiseStoreFromPaths(vegaPaths, true)
	if err != nil {
		cliLog.Error("Could not initialise wallets store", zap.Error(err))
		return fmt.Errorf("could not initialise wallets store: %w", err)
	}
	closer.Add(walletStore.Close)

	netStore, err := netStoreV1.InitialiseStore(vegaPaths)
	if err != nil {
		cliLog.Error("Could not initialise network store", zap.Error(err))
		return fmt.Errorf("could not initialise network store: %w", err)
	}

	svcStore, err := svcStoreV1.InitialiseStore(vegaPaths)
	if err != nil {
		cliLog.Error("Could not initialise service store", zap.Error(err))
		return fmt.Errorf("could not initialise service store: %w", err)
	}

	sessionStore, err := sessionStoreV1.InitialiseStore(vegaPaths)
	if err != nil {
		cliLog.Error("Could not initialise session store", zap.Error(err))
		return fmt.Errorf("could not initialise session store: %w", err)
	}

	var tokenStore connections.TokenStore
	if f.LoadTokens {
		cliLog.Warn("Long-living tokens enabled")
		p.Print(p.String().WarningBangMark().WarningText("Long-living tokens enabled").NextLine())
		s, err := v1.InitialiseStore(vegaPaths, f.tokensPassphrase)
		if err != nil {
			if errors.Is(err, walletapi.ErrWrongPassphrase) {
				return err
			}
			return fmt.Errorf("couldn't load the token store: %w", err)
		}
		closer.Add(s.Close)
		tokenStore = s
	} else {
		s := v1.NewEmptyStore()
		tokenStore = s
	}

	loggerBuilderFunc := func(levelName string) (*zap.Logger, zap.AtomicLevel, error) {
		svcLog, svcLogPath, level, err := buildJSONFileLogger(vegaPaths, paths.WalletServiceLogsHome, levelName)
		if err != nil {
			return nil, zap.AtomicLevel{}, err
		}

		p.Print(p.String().CheckMark().Text("Service logs located at: ").SuccessText(svcLogPath).NextLine())

		return svcLog, level, nil
	}

	consentRequests := make(chan serviceV1.ConsentRequest, MaxConsentRequests)
	sentTransactions := make(chan serviceV1.SentTransaction)
	closer.Add(func() {
		close(consentRequests)
		close(sentTransactions)
	})

	jobRunner := vgjob.NewRunner(context.Background())

	policy, err := buildPolicy(jobRunner.Ctx(), cliLog, p, f, consentRequests, sentTransactions)
	if err != nil {
		return err
	}

	receptionChanForParking := make(chan interactor.Interaction, 1000)
	closer.Add(func() {
		close(receptionChanForParking)
	})

	seqInteractor := interactor.NewParallelInteractor(jobRunner.Ctx(), receptionChanForParking)

	connectionsManager, err := connections.NewManager(serviceV2.NewStdTime(), walletStore, tokenStore, sessionStore, seqInteractor)
	if err != nil {
		return fmt.Errorf("could not create the connection manager: %w", err)
	}
	closer.Add(func() {
		connectionsManager.EndAllSessionConnections()
	})

	serviceStarter := service.NewStarter(walletStore, netStore, svcStore, connectionsManager, policy, seqInteractor, loggerBuilderFunc)

	rc, err := serviceStarter.Start(jobRunner, f.Network, f.NoVersionCheck)
	if err != nil {
		cliLog.Error("Failed to start HTTP server", zap.Error(err))
		jobRunner.StopAllJobs()
		return err
	}

	cliLog.Info("Starting HTTP service", zap.String("url", rc.ServiceURL))
	p.Print(p.String().CheckMark().Text("Starting HTTP service at: ").SuccessText(rc.ServiceURL).NextSection())

	receptionChanForFrontend := make(chan interactor.Interaction, 1000)
	closer.Add(func() {
		close(receptionChanForFrontend)
	})

	jobRunner.Go(func(jobCtx context.Context) {
		startInteractionParking(cliLog, jobCtx, receptionChanForParking, receptionChanForFrontend)
	})

	jobRunner.Go(func(jobCtx context.Context) {
		for {
			select {
			case <-jobCtx.Done():
				cliLog.Info("Stop listening to incoming interactions in front-end")
				return
			case interaction := <-receptionChanForFrontend:
				handleAPIv2Request(jobCtx, interaction, f.EnableAutomaticConsent, p)
			case consentRequest := <-consentRequests:
				handleAPIv1Request(consentRequest, cliLog, p, sentTransactions)
			}
		}
	})

	waitUntilInterruption(jobRunner.Ctx(), cliLog, p, rc.ErrCh)

	// Wait for all goroutine to exit.
	cliLog.Info("Waiting for the service to stop")
	p.Print(p.String().BlueArrow().Text("Waiting for the service to stop...").NextLine())
	jobRunner.StopAllJobs()
	cliLog.Info("The service stopped")
	p.Print(p.String().CheckMark().Text("The service stopped.").NextLine())

	return nil
}

func buildPolicy(ctx context.Context, cliLog *zap.Logger, p *printer.InteractivePrinter, f *RunServiceFlags, consentRequests chan serviceV1.ConsentRequest, sentTransactions chan serviceV1.SentTransaction) (serviceV1.Policy, error) {
	if vgterm.HasTTY() {
		cliLog.Info("TTY detected")
		if f.EnableAutomaticConsent {
			cliLog.Info("Automatic consent enabled")
			p.Print(p.String().WarningBangMark().WarningText("Automatic consent enabled").NextLine())
			return serviceV1.NewAutomaticConsentPolicy(), nil
		}
		cliLog.Info("Explicit consent enabled")
		p.Print(p.String().CheckMark().Text("Explicit consent enabled").NextLine())
		return serviceV1.NewExplicitConsentPolicy(ctx, consentRequests, sentTransactions), nil
	}

	cliLog.Info("No TTY detected")

	if !f.EnableAutomaticConsent {
		cliLog.Error("Explicit consent can't be used when no TTY is attached to the process")
		return nil, ErrEnableAutomaticConsentFlagIsRequiredWithoutTTY
	}

	cliLog.Info("Automatic consent enabled.")
	return serviceV1.NewAutomaticConsentPolicy(), nil
}

func buildJSONFileLogger(vegaPaths paths.Paths, logDir paths.StatePath, logLevel string) (*zap.Logger, string, zap.AtomicLevel, error) {
	loggerConfig := vgzap.DefaultConfig()
	loggerConfig = vgzap.WithFileOutputForDedicatedProcess(loggerConfig, vegaPaths.StatePathFor(logDir))
	logFilePath := loggerConfig.OutputPaths[0]
	loggerConfig = vgzap.WithJSONFormat(loggerConfig)
	loggerConfig = vgzap.WithLevel(loggerConfig, logLevel)

	level := loggerConfig.Level

	logger, err := vgzap.Build(loggerConfig)
	if err != nil {
		return nil, "", zap.AtomicLevel{}, fmt.Errorf("could not setup the logger: %w", err)
	}

	return logger, logFilePath, level, nil
}

// waitUntilInterruption will wait for a sigterm or sigint interrupt.
func waitUntilInterruption(ctx context.Context, cliLog *zap.Logger, p *printer.InteractivePrinter, errChan <-chan error) {
	gracefulStop := make(chan os.Signal, 1)
	defer func() {
		signal.Stop(gracefulStop)
		close(gracefulStop)
	}()

	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	for {
		select {
		case sig := <-gracefulStop:
			cliLog.Info("OS signal received", zap.String("signal", fmt.Sprintf("%+v", sig)))
			str := p.String()
			str.NextSection().WarningBangMark().WarningText(fmt.Sprintf("Signal \"%+v\" received.", sig)).NextLine()
			str.Pad().WarningText("You can hit CTRL+C once again to forcefully exit, but some resources may not be properly cleaned up.").NextSection()
			p.Print(str)
			return
		case err := <-errChan:
			cliLog.Error("Initiating shutdown due to an internal error reported by the service", zap.Error(err))
			return
		case <-ctx.Done():
			cliLog.Info("Stop listening to OS signals")
			return
		}
	}
}

func handleAPIv1Request(consentRequest serviceV1.ConsentRequest, log *zap.Logger, p *printer.InteractivePrinter, sentTransactions chan serviceV1.SentTransaction) {
	m := jsonpb.Marshaler{Indent: "    "}
	marshalledTx, err := m.MarshalToString(consentRequest.Tx)
	if err != nil {
		log.Error("could not marshal transaction from consent request", zap.Error(err))
		panic(err)
	}

	str := p.String()
	str.BlueArrow().Text("New transaction received: ").NextLine()
	str.InfoText(marshalledTx).NextLine()
	p.Print(str)

	if flags.DoYouApproveTx() {
		log.Info("user approved the signing of the transaction", zap.Any("transaction", marshalledTx))
		consentRequest.Confirmation <- serviceV1.ConsentConfirmation{Decision: true}
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
		consentRequest.Confirmation <- serviceV1.ConsentConfirmation{Decision: false}
		p.Print(p.String().DangerBangMark().DangerText("Transaction rejected").NextSection())
	}
}

func handleAPIv2Request(ctx context.Context, interaction interactor.Interaction, enableAutomaticConsent bool, p *printer.InteractivePrinter) {
	switch data := interaction.Data.(type) {
	case interactor.InteractionSessionBegan:
		p.Print(p.String().NextLine())
	case interactor.InteractionSessionEnded:
		p.Print(p.String().NextLine())
	case interactor.RequestWalletConnectionReview:
		p.Print(p.String().BlueArrow().Text("The application \"").InfoText(data.Hostname).Text("\" wants to connect to your wallet.").NextLine())
		var connectionApproval string
		approved, err := yesOrNo(ctx, data.ControlCh, p.String().QuestionMark().Text("Do you approve connecting your wallet to this application?"), p)
		if err != nil {
			p.Print(p.String().CrossMark().DangerText(err.Error()).NextLine())
			return
		}
		if approved {
			p.Print(p.String().CheckMark().Text("Connection approved.").NextLine())
			connectionApproval = string(preferences.ApprovedOnlyThisTime)
		} else {
			p.Print(p.String().CrossMark().Text("Connection rejected.").NextLine())
			connectionApproval = string(preferences.RejectedOnlyThisTime)
		}
		data.ResponseCh <- interactor.Interaction{
			TraceID: interaction.TraceID,
			Name:    interactor.WalletConnectionDecisionName,
			Data: interactor.WalletConnectionDecision{
				ConnectionApproval: connectionApproval,
			},
		}
	case interactor.RequestWalletSelection:
		str := p.String().BlueArrow().Text("Here are the available wallets:").NextLine()
		for _, w := range data.AvailableWallets {
			str.ListItem().Text("- ").InfoText(w).NextLine()
		}
		p.Print(str)
		selectedWallet, err := readInput(ctx, data.ControlCh, p.String().QuestionMark().Text("Which wallet do you want to use? "), p, data.AvailableWallets)
		if err != nil {
			p.Print(p.String().CrossMark().DangerText(err.Error()).NextLine())
			return
		}
		data.ResponseCh <- interactor.Interaction{
			TraceID: interaction.TraceID,
			Name:    interactor.SelectedWalletName,
			Data: interactor.SelectedWallet{
				Wallet: selectedWallet,
			},
		}
	case interactor.RequestPassphrase:
		if len(data.Reason) != 0 {
			str := p.String().BlueArrow().Text(data.Reason).NextLine()
			p.Print(str)
		}
		passphrase, err := readPassphrase(ctx, data.ControlCh, p.String().BlueArrow().Text("Enter the passphrase for the wallet \"").InfoText(data.Wallet).Text("\": "), p)
		if err != nil {
			p.Print(p.String().CrossMark().DangerText(err.Error()).NextLine())
			return
		}
		data.ResponseCh <- interactor.Interaction{
			TraceID: interaction.TraceID,
			Name:    interactor.EnteredPassphraseName,
			Data: interactor.EnteredPassphrase{
				Passphrase: passphrase,
			},
		}
	case interactor.ErrorOccurred:
		if data.Type == string(walletapi.InternalErrorType) {
			str := p.String().DangerBangMark().DangerText("An internal error occurred: ").DangerText(data.Error).NextLine()
			str.DangerBangMark().DangerText("The request has been canceled.").NextLine()
			p.Print(str)
		} else if data.Type == string(walletapi.UserErrorType) {
			p.Print(p.String().DangerBangMark().DangerText(data.Error).NextLine())
		} else {
			p.Print(p.String().DangerBangMark().DangerText(fmt.Sprintf("Error: %s (%s)", data.Error, data.Type)).NextLine())
		}
	case interactor.Log:
		str := p.String()
		switch data.Type {
		case string(walletapi.InfoLog):
			str.BlueArrow()
		case string(walletapi.ErrorLog):
			str.CrossMark()
		case string(walletapi.WarningLog):
			str.WarningBangMark()
		case string(walletapi.SuccessLog):
			str.CheckMark()
		default:
			str.Text("- ")
		}
		p.Print(str.Text(data.Message).NextLine())
	case interactor.RequestSucceeded:
		if data.Message == "" {
			p.Print(p.String().CheckMark().SuccessText("Request succeeded").NextLine())
		} else {
			p.Print(p.String().CheckMark().SuccessText(data.Message).NextLine())
		}
	case interactor.RequestPermissionsReview:
		str := p.String().BlueArrow().Text("The application \"").InfoText(data.Hostname).Text("\" requires the following permissions for \"").InfoText(data.Wallet).Text("\":").NextLine()
		for perm, access := range data.Permissions {
			str.ListItem().Text("- ").InfoText(perm).Text(": ").InfoText(access).NextLine()
		}
		p.Print(str)
		approved, err := yesOrNo(ctx, data.ControlCh, p.String().QuestionMark().Text("Do you want to grant these permissions?"), p)
		if err != nil {
			p.Print(p.String().CrossMark().DangerText(err.Error()).NextLine())
			return
		}
		if approved {
			p.Print(p.String().CheckMark().Text("Permissions update approved.").NextLine())
		} else {
			p.Print(p.String().CrossMark().Text("Permissions update rejected.").NextLine())
		}
		data.ResponseCh <- interactor.Interaction{
			TraceID: interaction.TraceID,
			Name:    interactor.DecisionName,
			Data: interactor.Decision{
				Approved: approved,
			},
		}
	case interactor.RequestTransactionReviewForSending:
		str := p.String().BlueArrow().Text("The application \"").InfoText(data.Hostname).Text("\" wants to send the following transaction:").NextLine()
		str.Pad().Text("Using the key: ").InfoText(data.PublicKey).NextLine()
		str.Pad().Text("From the wallet: ").InfoText(data.Wallet).NextLine()
		fmtCmd := strings.Replace("  "+data.Transaction, "\n", "\n  ", -1)
		str.InfoText(fmtCmd).NextLine()
		p.Print(str)
		approved := true
		if enableAutomaticConsent {
			p.Print(p.String().CheckMark().Text("Sending automatically approved.").NextLine())
		} else {
			a, err := yesOrNo(ctx, data.ControlCh, p.String().QuestionMark().Text("Do you want to send this transaction?"), p)
			if err != nil {
				p.Print(p.String().CrossMark().DangerText(err.Error()).NextLine())
				return
			}
			approved = a
			if approved {
				p.Print(p.String().CheckMark().Text("Sending approved.").NextLine())
			} else {
				p.Print(p.String().CrossMark().Text("Sending rejected.").NextLine())
			}
		}
		data.ResponseCh <- interactor.Interaction{
			TraceID: interaction.TraceID,
			Name:    interactor.DecisionName,
			Data: interactor.Decision{
				Approved: approved,
			},
		}
	case interactor.RequestTransactionReviewForSigning:
		str := p.String().BlueArrow().Text("The application \"").InfoText(data.Hostname).Text("\" wants to sign the following transaction:").NextLine()
		str.Pad().Text("Using the key: ").InfoText(data.PublicKey).NextLine()
		str.Pad().Text("From the wallet: ").InfoText(data.Wallet).NextLine()
		fmtCmd := strings.Replace("  "+data.Transaction, "\n", "\n  ", -1)
		str.InfoText(fmtCmd).NextLine()
		p.Print(str)
		approved := true
		if enableAutomaticConsent {
			p.Print(p.String().CheckMark().Text("Signing automatically approved.").NextLine())
		} else {
			a, err := yesOrNo(ctx, data.ControlCh, p.String().QuestionMark().Text("Do you want to sign this transaction?"), p)
			if err != nil {
				p.Print(p.String().CrossMark().DangerText(err.Error()).NextLine())
				return
			}
			approved = a
			if approved {
				p.Print(p.String().CheckMark().Text("Signing approved.").NextLine())
			} else {
				p.Print(p.String().CrossMark().Text("Signing rejected.").NextLine())
			}
		}
		data.ResponseCh <- interactor.Interaction{
			TraceID: interaction.TraceID,
			Name:    interactor.DecisionName,
			Data: interactor.Decision{
				Approved: approved,
			},
		}
	case interactor.RequestTransactionReviewForChecking:
		str := p.String().BlueArrow().Text("The application \"").InfoText(data.Hostname).Text("\" wants to check the following transaction:").NextLine()
		str.Pad().Text("Using the key: ").InfoText(data.PublicKey).NextLine()
		str.Pad().Text("From the wallet: ").InfoText(data.Wallet).NextLine()
		fmtCmd := strings.Replace("  "+data.Transaction, "\n", "\n  ", -1)
		str.InfoText(fmtCmd).NextLine()
		p.Print(str)
		approved := true
		if enableAutomaticConsent {
			p.Print(p.String().CheckMark().Text("Checking automatically approved.").NextLine())
		} else {
			a, err := yesOrNo(ctx, data.ControlCh, p.String().QuestionMark().Text("Do you allow the network to check this transaction?"), p)
			if err != nil {
				p.Print(p.String().CrossMark().DangerText(err.Error()).NextLine())
				return
			}
			approved = a
			if approved {
				p.Print(p.String().CheckMark().Text("Checking approved.").NextLine())
			} else {
				p.Print(p.String().CrossMark().Text("Checking rejected.").NextLine())
			}
		}
		data.ResponseCh <- interactor.Interaction{
			TraceID: interaction.TraceID,
			Name:    interactor.DecisionName,
			Data: interactor.Decision{
				Approved: approved,
			},
		}
	case interactor.TransactionFailed:
		str := p.String()
		str.DangerBangMark().DangerText("The transaction failed.").NextLine()
		str.Pad().DangerText(data.Error.Error()).NextLine()
		str.Pad().Text("Sent at: ").Text(data.SentAt.Format(time.ANSIC)).NextLine()
		p.Print(str)
	case interactor.TransactionSucceeded:
		str := p.String()
		str.CheckMark().SuccessText("The transaction has been delivered.").NextLine()
		str.Pad().Text("Transaction hash: ").SuccessText(data.TxHash).NextLine()
		str.Pad().Text("Sent at: ").Text(data.SentAt.Format(time.ANSIC)).NextLine()
		p.Print(str)
	default:
		panic(fmt.Sprintf("unhandled interaction: %q", interaction.Name))
	}
}

func readInput(ctx context.Context, controlCh chan error, question *printer.FormattedString, p *printer.InteractivePrinter, options []string) (string, error) {
	inputCh := make(chan string)
	defer close(inputCh)

	reader, err := cancelreader.NewReader(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("could not initialize the input reader: %w", err)
	}
	defer reader.Cancel()

	go func() {
		for {
			p.Print(question)

			answer, err := readString(reader)
			if err != nil {
				return
			}

			if len(options) == 0 {
				inputCh <- answer
				return
			}
			for _, option := range options {
				if answer == option {
					inputCh <- answer
					return
				}
			}
			if len(answer) > 0 {
				p.Print(p.String().DangerBangMark().DangerText(fmt.Sprintf("%q is not a valid option", answer)).NextLine())
			}
		}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-controlCh:
		reader.Cancel()
		return "", err
	case input := <-inputCh:
		return input, nil
	}
}

func yesOrNo(ctx context.Context, controlCh <-chan error, question *printer.FormattedString, p *printer.InteractivePrinter) (bool, error) {
	choiceCh := make(chan bool)
	defer close(choiceCh)

	reader, err := cancelreader.NewReader(os.Stdin)
	if err != nil {
		return false, fmt.Errorf("could not initialize the input reader: %w", err)
	}
	defer reader.Cancel()

	go func() {
		question.Text(" (yes/no) ")

		for {
			p.Print(question)

			answer, err := readString(reader)
			if err != nil {
				return
			}

			switch answer {
			case "yes", "y":
				choiceCh <- true
				return
			case "no", "n":
				choiceCh <- false
				return
			default:
				if len(answer) > 0 {
					p.Print(p.String().DangerBangMark().DangerText(fmt.Sprintf("%q is not a valid answer, enter \"yes\" or \"no\"\n", answer)))
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case err := <-controlCh:
		reader.Cancel()
		return false, err
	case choice, ok := <-choiceCh:
		return ok && choice, nil
	}
}

func readString(reader cancelreader.CancelReader) (string, error) {
	var line string
	for {
		var input [1024]byte
		_, err := reader.Read(input[:])
		if err != nil {
			return "", err
		}
		index := bytes.IndexByte(input[:], '\n')
		line += string(input[:index])
		if index != -1 {
			break
		}
	}

	line = strings.ToLower(strings.Trim(line, " \r\n\t"))
	return line, nil
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

func readPassphrase(ctx context.Context, controlCh chan error, question *printer.FormattedString, p *printer.InteractivePrinter) (string, error) {
	inputCh := make(chan string)
	defer close(inputCh)

	// We cannot interrupt cleanly an on-going password read. So, at least, we
	// ensure it can stop on the next password attempt.
	shouldStop := atomic.Bool{}
	waitForExitInput := make(chan interface{})

	go func() {
		for {
			p.Print(question)
			passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				panic(fmt.Errorf("could not read passphrase: %w", err))
			}
			p.Print(p.String().NextLine())
			if shouldStop.Load() {
				close(waitForExitInput)
				return
			}
			if len(passphrase) > 0 {
				inputCh <- string(passphrase)
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-controlCh:
		shouldStop.Store(true)
		<-waitForExitInput
		return "", err
	case input := <-inputCh:
		return input, nil
	}
}

func startInteractionParking(log *zap.Logger, ctx context.Context, inboundCh <-chan interactor.Interaction, outboundCh chan<- interactor.Interaction) {
	sessionsOrder := []string{}
	parkedInteractionSessions := map[string]chan interactor.Interaction{}

	defer func() {
		for _, iChan := range parkedInteractionSessions {
			close(iChan)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stop listening to incoming interactions in parking")
			return
		case interaction, ok := <-inboundCh:
			if !ok {
				return
			}

			if len(sessionsOrder) == 0 {
				sessionsOrder = append(sessionsOrder, interaction.TraceID)
			}

			// If the interaction we receive is from the session currently
			// handled in the frontend, we transmit it immediately.
			if sessionsOrder[0] == interaction.TraceID {
				outboundCh <- interaction
				// If this is the last interaction for the current session, we
				// free up the resources and transmit the next session to the UI.
				if _, ok := interaction.Data.(interactor.InteractionSessionEnded); ok {
					sessionsOrder = switchToNextSession(sessionsOrder, parkedInteractionSessions, outboundCh)
				}
			} else {
				// If not, then we park it until the current session end.
				parkedSessionCh, ok := parkedInteractionSessions[interaction.TraceID]
				if !ok {
					// First time we see this session, we track it.
					parkedSessionCh = make(chan interactor.Interaction, 100)
					parkedInteractionSessions[interaction.TraceID] = parkedSessionCh
					sessionsOrder = append(sessionsOrder, interaction.TraceID)
				}
				parkedSessionCh <- interaction
			}
		}
	}
}

func switchToNextSession(sessionsOrder []string, parkedInteractionSessions map[string]chan interactor.Interaction, outboundCh chan<- interactor.Interaction) []string {
	// Pop this session out the queue, and move onto the next
	// session.
	sessionsOrder = sessionsOrder[1:]

	if len(sessionsOrder) == 0 {
		return sessionsOrder
	}

	currentSessionCh := parkedInteractionSessions[sessionsOrder[0]]
	hasInteractionsToSend := true
	for hasInteractionsToSend {
		select {
		case currentSessionInteraction, ok := <-currentSessionCh:
			if !ok {
				hasInteractionsToSend = false
				break
			}
			outboundCh <- currentSessionInteraction
			if _, ok := currentSessionInteraction.Data.(interactor.InteractionSessionEnded); ok {
				// We remove the session and its interactions buffer from the
				// parked ones, because the next interactions we will receive
				// for that session will be transmitted immediately.
				close(currentSessionCh)
				delete(parkedInteractionSessions, sessionsOrder[0])

				// The session is already finished, move to the next until we
				// transmitted all parked session or until a session is ongoing.
				return switchToNextSession(sessionsOrder, parkedInteractionSessions, outboundCh)
			}
		default:
			hasInteractionsToSend = false
		}
	}

	// We remove the session and its interactions buffer from the
	// parked ones, because the next interactions we will receive
	// for that session will be transmitted immediately.
	close(currentSessionCh)
	delete(parkedInteractionSessions, sessionsOrder[0])
	return sessionsOrder
}
