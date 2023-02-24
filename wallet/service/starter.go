package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	vgclose "code.vegaprotocol.io/vega/libs/close"
	vgjob "code.vegaprotocol.io/vega/libs/job"
	vgzap "code.vegaprotocol.io/vega/libs/zap"
	coreversion "code.vegaprotocol.io/vega/version"
	"code.vegaprotocol.io/vega/wallet/api"
	nodeapi "code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/api/spam"
	"code.vegaprotocol.io/vega/wallet/network"
	"code.vegaprotocol.io/vega/wallet/node"
	servicev1 "code.vegaprotocol.io/vega/wallet/service/v1"
	servicev2 "code.vegaprotocol.io/vega/wallet/service/v2"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	walletversion "code.vegaprotocol.io/vega/wallet/version"
	"code.vegaprotocol.io/vega/wallet/wallets"
	"go.uber.org/zap"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/wallet/service NetworkStore

const serviceStoppingTimeout = 3 * time.Minute

var ErrCannotStartMultipleServiceAtTheSameTime = errors.New("cannot start multiple service at the same time")

// PolicyBuilderFunc return the policy the API v1.
type PolicyBuilderFunc func(ctx context.Context) servicev1.Policy

// InteractorBuilderFunc returns the interactor to use in the client API.
type InteractorBuilderFunc func(ctx context.Context) api.Interactor

// LoggerBuilderFunc is used to build a logger. It returns the built logger and a
// zap.AtomicLevel to allow the caller to dynamically change the log level.
type LoggerBuilderFunc func(level string) (*zap.Logger, zap.AtomicLevel, error)

type ConnectionsManagerBuilderFunc func() *connections.Manager

type ProcessStoppedNotifier func()

type NetworkStore interface {
	NetworkExists(string) (bool, error)
	GetNetwork(string) (*network.Network, error)
}

type Starter struct {
	walletStore api.WalletStore
	netStore    NetworkStore
	svcStore    Store

	connectionsManager    *connections.Manager
	policyBuilderFunc     PolicyBuilderFunc
	interactorBuilderFunc InteractorBuilderFunc
	loggerBuilderFunc     LoggerBuilderFunc

	isStarted atomic.Bool
}

func (s *Starter) Start(jobRunner *vgjob.Runner, network string, noVersionCheck bool) (_ string, _ <-chan error, err error) {
	if s.isStarted.Load() {
		return "", nil, ErrCannotStartMultipleServiceAtTheSameTime
	}
	s.isStarted.Store(true)
	defer func() {
		if err != nil {
			// If we exit with an error, we reset the state.
			s.isStarted.Store(false)
		}
	}()

	logger, logLevel, errDetails := s.buildServiceLogger(network)
	if errDetails != nil {
		return "", nil, errDetails
	}
	defer vgzap.Sync(logger)

	serviceCfg, err := s.svcStore.GetConfig()
	if err != nil {
		return "", nil, fmt.Errorf("could not retrieve the service configuration: %w", err)
	}

	if err := serviceCfg.Validate(); err != nil {
		return "", nil, err
	}

	// Since we successfully retrieve the service configuration, we can update
	// the log level to the specified one.
	if err := updateLogLevel(logLevel, serviceCfg); err != nil {
		return "", nil, err
	}

	networkCfg, err := s.networkConfig(logger, network)
	if err != nil {
		return "", nil, err
	}

	if !noVersionCheck {
		if err := s.ensureSoftwareIsCompatibleWithNetwork(logger, networkCfg); err != nil {
			return "", nil, err
		}
	} else {
		logger.Warn("The compatibility check between the software and the network has been skipped")
	}

	if err := s.ensureServiceIsInitialised(logger); err != nil {
		return "", nil, err
	}

	// Check if the port we want to bind is free. It's not fool-proof, but it
	// should catch most of the port-binding problems.
	if err := ensurePortCanBeBound(jobRunner.Ctx(), logger, serviceCfg.Server.String()); err != nil {
		return "", nil, err
	}

	apiLogger := logger.Named("api")

	// We have several components that hold resources that needs to be released
	// when stopping the service.
	closer := vgclose.NewCloser()

	proofOfWork := spam.NewHandler()

	// API v1
	apiV1, err := s.buildAPIV1(jobRunner.Ctx(), apiLogger, networkCfg, serviceCfg, proofOfWork, closer)
	if err != nil {
		logger.Error("Could not build the HTTP API v1", zap.Error(err))
		return "", nil, err
	}

	// API v2
	apiV2, err := s.buildAPIV2(jobRunner.Ctx(), apiLogger, networkCfg, proofOfWork, closer)
	if err != nil {
		logger.Error("Could not build the HTTP API v2", zap.Error(err))
		return "", nil, err
	}

	svc := NewService(logger.Named("http-server"), serviceCfg, apiV1, apiV2)

	// This job is responsible for stopping the service when the job context is
	// set as done.
	// This is required because we can't bind the service to a context.
	jobRunner.Go(func(jobCtx context.Context) {
		defer s.isStarted.Store(false)
		defer vgzap.Sync(logger)

		// We wait for the job context to be cancelled to stop the service.
		<-jobCtx.Done()

		// Stopping the service with a maximum wait of 3 minutes.
		ctxWithTimeout, cancelFunc := context.WithTimeout(context.Background(), serviceStoppingTimeout)
		defer cancelFunc()
		if err := svc.Stop(ctxWithTimeout); err != nil {
			logger.Warn("Could not properly stop the HTTP server",
				zap.Duration("timeout", serviceStoppingTimeout),
				zap.Error(err),
			)
		} else {
			logger.Warn("the HTTP server gracefully stopped")
		}
	})

	internalErrorReporter := make(chan error, 1)

	jobRunner.Go(func(_ context.Context) {
		defer close(internalErrorReporter)
		defer vgzap.Sync(logger)

		logger.Info("Starting the HTTP server")
		if err := svc.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Error while running HTTP server", zap.Error(err))
			// We warn the caller about the error, so it know something went wrong
			// with the service and can cancel the service.
			internalErrorReporter <- err
		}

		// Freeing associated components.
		closer.CloseAll()

		logger.Info("The service exited")
	})

	return serviceCfg.Server.String(), internalErrorReporter, nil
}

// buildAPIV1
// This API is deprecated.
func (s *Starter) buildAPIV1(ctx context.Context, logger *zap.Logger, networkCfg *network.Network, serviceCfg *Config, spam *spam.Handler, closer *vgclose.Closer) (*servicev1.API, error) {
	apiV1Logger := logger.Named("v1")

	forwarder, err := node.NewForwarder(apiV1Logger.Named("forwarder"), networkCfg.API.GRPC)
	if err != nil {
		logger.Error("Could not initialise the node forwarder", zap.Error(err))
		return nil, fmt.Errorf("could not initialise the node forwarder: %w", err)
	}
	// Don't forget to stop all connections to the nodes.
	closer.Add(forwarder.Stop)

	auth, err := servicev1.NewAuth(apiV1Logger.Named("auth"), s.svcStore, serviceCfg.APIV1.MaximumTokenDuration.Get())
	if err != nil {
		logger.Error("Could not initialise the authentication layer", zap.Error(err))
		return nil, fmt.Errorf("could not initialise the authentication layer: %w", err)
	}
	// Don't forget to close the sessions.
	closer.Add(auth.RevokeAllToken)

	// We don't close/stop the policy ourselves, this should be done by the provider
	// of the builder function. We don't close what we don't own.
	policy := s.policyBuilderFunc(ctx)

	handler := wallets.NewHandler(s.walletStore)

	return servicev1.NewAPI(apiV1Logger, handler, auth, forwarder, policy, networkCfg, spam), nil
}

func (s *Starter) buildAPIV2(ctx context.Context, logger *zap.Logger, cfg *network.Network, pow api.SpamHandler, closer *vgclose.Closer) (*servicev2.API, error) {
	apiV2logger := logger.Named("v2")
	clientAPILogger := apiV2logger.Named("client-api")

	nodeSelector, err := nodeapi.BuildRoundRobinSelectorWithRetryingNodes(clientAPILogger, cfg.API.GRPC.Hosts, cfg.API.GRPC.Retries)
	if err != nil {
		logger.Error("Could not build the node selector", zap.Error(err))
		return nil, err
	}
	closer.Add(nodeSelector.Stop)

	// We don't close the interactor ourselves, this should be done by
	// the provider of the builder function. We don't close what we don't own.
	interactor := s.interactorBuilderFunc(ctx)

	clientAPI, err := api.BuildClientAPI(s.walletStore, interactor, nodeSelector, pow)
	if err != nil {
		logger.Error("Could not instantiate the client part of the JSON-RPC API", zap.Error(err))
		return nil, fmt.Errorf("could not instantiate the client part of the JSON-RPC API: %w", err)
	}

	return servicev2.NewAPI(apiV2logger, clientAPI, s.connectionsManager), nil
}

func (s *Starter) buildServiceLogger(network string) (*zap.Logger, zap.AtomicLevel, error) {
	// We set the logger with the "INFO" level by default. It will be changed once
	// we get to retrieve the log level from the network configuration.
	logger, level, err := s.loggerBuilderFunc("info")
	if err != nil {
		return nil, zap.AtomicLevel{}, err
	}

	logger = logger.
		Named("service").
		With(zap.String("network", network))

	return logger, level, nil
}

func (s *Starter) ensureSoftwareIsCompatibleWithNetwork(logger *zap.Logger, networkCfg *network.Network) error {
	networkVersion, err := walletversion.GetNetworkVersionThroughGRPC(networkCfg.API.GRPC.Hosts)
	if err != nil {
		logger.Error("Could not verify the compatibility between the network and the software", zap.Error(err))
		return fmt.Errorf("could not verify the compatibility between the network and the software: %w", err)
	}

	coreVersion := coreversion.Get()

	if networkVersion != coreVersion {
		logger.Error("This software is not compatible with the network",
			zap.String("network-version", networkVersion),
			zap.String("core-version", coreVersion),
		)
		return fmt.Errorf("this software is not compatible with this network as the network is running version %s but this software expects the version %s", networkVersion, coreversion.Get())
	}

	logger.Info("This software is compatible with the network")

	return nil
}

func (s *Starter) networkConfig(logger *zap.Logger, network string) (*network.Network, error) {
	exists, err := s.netStore.NetworkExists(network)
	if err != nil {
		logger.Error("Could not verify the network existence", zap.Error(err))
		return nil, fmt.Errorf("could not verify the network existence: %w", err)
	}
	if !exists {
		logger.Error("The requested network does not exists", zap.String("network", network))
		return nil, api.ErrNetworkDoesNotExist
	}

	networkCfg, err := s.netStore.GetNetwork(network)
	if err != nil {
		logger.Error("Could not retrieve the network configuration", zap.Error(err))
		return nil, fmt.Errorf("could not retrieve the network configuration: %w", err)
	}

	if err := networkCfg.EnsureCanConnectGRPCNode(); err != nil {
		logger.Error("The requested network can't connect to the nodes gRPC API", zap.Error(err), zap.String("network", network))
		return nil, err
	}

	logger.Info("The network configuration has been loaded", zap.String("network", network))

	return networkCfg, nil
}

func (s *Starter) ensureServiceIsInitialised(logger *zap.Logger) error {
	if isInit, err := IsInitialised(s.svcStore); err != nil {
		logger.Error("Could not verify if the service is properly running", zap.Error(err))
		return fmt.Errorf("could not verify if the service is properly initialised: %w", err)
	} else if !isInit {
		logger.Info("The service is not initialise")
		if err = InitialiseService(s.svcStore, false); err != nil {
			logger.Error("Could not initialise the service", zap.Error(err))
			return fmt.Errorf("could not initialise the service: %w", err)
		}
		logger.Info("The service has been initialised")
	} else {
		logger.Info("The service has already been initialised")
	}
	return nil
}

func updateLogLevel(logLevel zap.AtomicLevel, serviceCfg *Config) error {
	parsedLevel, err := zap.ParseAtomicLevel(serviceCfg.LogLevel.String())
	if err != nil {
		return fmt.Errorf("invalid log level specified in the service configuration: %w", err)
	}
	logLevel.SetLevel(parsedLevel.Level())
	return nil
}

func NewStarter(
	walletStore api.WalletStore,
	netStore api.NetworkStore,
	svcStore Store,
	connectionsManager *connections.Manager,
	policyBuilderFunc PolicyBuilderFunc,
	interactorBuilderFunc InteractorBuilderFunc,
	loggerBuilderFunc LoggerBuilderFunc,
) *Starter {
	return &Starter{
		walletStore:           walletStore,
		netStore:              netStore,
		svcStore:              svcStore,
		connectionsManager:    connectionsManager,
		policyBuilderFunc:     policyBuilderFunc,
		interactorBuilderFunc: interactorBuilderFunc,
		loggerBuilderFunc:     loggerBuilderFunc,
	}
}

func ensurePortCanBeBound(ctx context.Context, logger *zap.Logger, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, bytes.NewReader([]byte{}))
	if err != nil {
		logger.Error("Could not build the request verifying the state of the port to bind", zap.Error(err))
		return fmt.Errorf("could not build the request verifying the state of the port to bind: %w", err)
	}

	response, err := http.DefaultClient.Do(req)
	if err == nil {
		// If there is no error, it means the server managed to establish a
		// connection of some kind, whereas we would have liked it to be unable
		// to connect to anything, which would have implied this host is free to
		// use.
		logger.Error("Could not start the service as an application is already served on that url", zap.String("url", url))
		return fmt.Errorf("could not start the service as an application is already served on %q", url)
	}
	defer func() {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
	}()

	logger.Info("The URL seems available for use")
	return nil
}
