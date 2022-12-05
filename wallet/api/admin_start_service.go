package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgzap "code.vegaprotocol.io/vega/libs/zap"
	"code.vegaprotocol.io/vega/paths"
	coreversion "code.vegaprotocol.io/vega/version"
	nodeapi "code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/network"
	"code.vegaprotocol.io/vega/wallet/node"
	"code.vegaprotocol.io/vega/wallet/service"
	walletversion "code.vegaprotocol.io/vega/wallet/version"
	"code.vegaprotocol.io/vega/wallet/wallets"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

type ProcessStoppedNotifier func()

type AdminStartServiceParams struct {
	Network        string `json:"network"`
	NoVersionCheck bool   `json:"noVersionCheck"`
}

type AdminStartServiceResult struct {
	URL string `json:"url"`
}

type AdminStartService struct {
	walletStore     WalletStore
	netStore        NetworkStore
	svcStore        ServiceStore
	servicesManager *ServicesManager

	policyBuilderFunc         PolicyBuilderFunc
	interactorBuilderFunc     InteractorBuilderFunc
	loggerBuilderFunc         LoggerBuilderFunc
	shutdownSwitchBuilderFunc ShutdownSwitchBuilder
}

// Handle create a transaction to rotate the keys.
//
// Note: We do not use the context passed as parameter as we don't want to tie
// a short living context to the long-running goroutines, like with an HTTP request
// context. To manage the lifetime of the goroutines, the context builder should
// be used to wrap the parent context.
func (h *AdminStartService) Handle(_ context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminStartServiceParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	logger, logLevel, errDetails := h.buildServiceLogger(params.Network)
	if errDetails != nil {
		return nil, errDetails
	}
	defer vgzap.Sync(logger)

	networkCfg, errDetails := h.networkConfig(logger, params.Network)
	if errDetails != nil {
		return nil, errDetails
	}

	// Since we successfully retrieve the network config, we can update the log
	// level to the specified one.
	if errDetails := updateLogLevel(logLevel, networkCfg); errDetails != nil {
		return nil, errDetails
	}

	if !params.NoVersionCheck {
		if errDetails := h.ensureSoftwareIsCompatibleWithNetwork(logger, networkCfg); errDetails != nil {
			return nil, errDetails
		}
	} else {
		logger.Warn("The compatibility check between the software and the network has been skipped")
	}

	if errDetails := h.ensureServiceIsInitialised(logger); errDetails != nil {
		return nil, errDetails
	}

	svcURL := fmt.Sprintf("%s:%v", networkCfg.Host, networkCfg.Port)

	shutdownSwitch := h.shutdownSwitchBuilderFunc()

	// Check if the port we want to bind is free. It's not fool-proof, but it
	// should catch most of the port-binding problems.
	if errDetails := ensurePortCanBeBound(shutdownSwitch.Ctx(), logger, svcURL); errDetails != nil {
		return nil, errDetails
	}

	// API v1
	// This API is deprecated.
	apiV1Logger := logger.Named("api-v1")

	auth, err := service.NewAuth(apiV1Logger.Named("auth"), h.svcStore, networkCfg.TokenExpiry.Get())
	if err != nil {
		logger.Error("Could not initialise the authentication layer", zap.Error(err))
		return nil, internalError(fmt.Errorf("could not initialise the authentication layer: %w", err))
	}

	forwarder, err := node.NewForwarder(apiV1Logger.Named("forwarder"), networkCfg.API.GRPC)
	if err != nil {
		logger.Error("Could not initialise the node forwarder", zap.Error(err))
		return nil, internalError(fmt.Errorf("could not initialise the node forwarder: %w", err))
	}

	policy := h.policyBuilderFunc(shutdownSwitch.Ctx())

	handler := wallets.NewHandler(h.walletStore)

	// API v2
	sessions := session.NewSessions()
	clientAPI, err := h.buildClientAPI(shutdownSwitch.Ctx(), logger, networkCfg, sessions)
	if err != nil {
		logger.Error("Could not build the client JSON-RPC API", zap.Error(err))
		return nil, internalError(err)
	}

	svc := service.NewService(logger.Named("http-server"), networkCfg, clientAPI, handler, auth, forwarder, policy)

	shutdownSwitch.BeforeCancelFunc(func() {
		if err := svc.Stop(); err != nil {
			logger.Warn("Could not properly stop the HTTP server", zap.Error(err))
		}
	})

	if err := h.servicesManager.RegisterService(params.Network, svcURL, sessions, shutdownSwitch); err != nil {
		logger.Error("Could not register the service", zap.Error(err))
		return nil, internalError(fmt.Errorf("could not register the service: %w", err))
	}

	notifyServiceStopped := shutdownSwitch.BindToProcess()
	go func() {
		if err := h.startService(svc, logger); err != nil {
			h.servicesManager.AbortService(params.Network, err)
		} else {
			h.servicesManager.StopService(params.Network)
		}
		// Will be call when the previous blocking call returns.
		notifyServiceStopped()
		vgzap.Sync(logger)
	}()

	return AdminStartServiceResult{
		URL: svcURL,
	}, nil
}

func updateLogLevel(logLevel zap.AtomicLevel, networkCfg *network.Network) *jsonrpc.ErrorDetails {
	parsedLevel, err := zap.ParseAtomicLevel(networkCfg.LogLevel.String())
	if err != nil {
		return internalError(fmt.Errorf("invalid log level specified in the network configuration: %w", err))
	}
	logLevel.SetLevel(parsedLevel.Level())
	return nil
}

func (h *AdminStartService) buildServiceLogger(network string) (*zap.Logger, zap.AtomicLevel, *jsonrpc.ErrorDetails) {
	// We set the logger with the "INFO" level by default. It will be changed once
	// we get to retrieve the log level from the network configuration.
	logger, level, err := h.loggerBuilderFunc(paths.WalletServiceLogsHome, "info")
	if err != nil {
		return nil, zap.AtomicLevel{}, internalError(err)
	}

	logger = logger.
		Named("service").
		With(zap.String("network", network))

	return logger, level, nil
}

func (h *AdminStartService) ensureSoftwareIsCompatibleWithNetwork(logger *zap.Logger, networkCfg *network.Network) *jsonrpc.ErrorDetails {
	networkVersion, err := walletversion.GetNetworkVersionThroughGRPC(networkCfg.API.GRPC.Hosts)
	if err != nil {
		logger.Error("Could not verify the compatibility between the network and the software", zap.Error(err))
		return internalError(fmt.Errorf("could not verify the compatibility between the network and the software: %w", err))
	}

	coreVersion := coreversion.Get()

	if networkVersion != coreVersion {
		logger.Error("This software is not compatible with the network",
			zap.String("network-version", networkVersion),
			zap.String("core-version", coreVersion),
		)
		return incompatibilityBetweenSoftwareAndNetworkError(networkVersion)
	}

	logger.Info("This software is compatible with the network")

	return nil
}

func (h *AdminStartService) networkConfig(logger *zap.Logger, network string) (*network.Network, *jsonrpc.ErrorDetails) {
	exists, err := h.netStore.NetworkExists(network)
	if err != nil {
		logger.Error("Could not verify the network existence", zap.Error(err))
		return nil, internalError(fmt.Errorf("could not verify the network existence: %w", err))
	}
	if !exists {
		logger.Error("The requested network does not exists", zap.String("network", network))
		return nil, invalidParams(ErrNetworkDoesNotExist)
	}

	networkCfg, err := h.netStore.GetNetwork(network)
	if err != nil {
		logger.Error("Could not retrieve the network configuration", zap.Error(err))
		return nil, internalError(fmt.Errorf("could not retrieve the network configuration: %w", err))
	}

	if err := networkCfg.EnsureCanConnectGRPCNode(); err != nil {
		logger.Error("The requested network can't connect to the nodes gRPC API", zap.Error(err), zap.String("network", network))
		return nil, invalidParams(err)
	}

	logger.Info("The network configuration has been loaded", zap.String("network", network))

	return networkCfg, nil
}

func (h *AdminStartService) ensureServiceIsInitialised(logger *zap.Logger) *jsonrpc.ErrorDetails {
	if isInit, err := service.IsInitialised(h.svcStore); err != nil {
		logger.Error("Could not verify if the service is properly running", zap.Error(err))
		return internalError(fmt.Errorf("could not verify if the service is properly initialised: %w", err))
	} else if !isInit {
		logger.Info("The service is not initialise")
		if err = service.InitialiseService(h.svcStore, false); err != nil {
			logger.Error("Could not initialise the service", zap.Error(err))
			return internalError(fmt.Errorf("could not initialise the service: %w", err))
		}
		logger.Info("The service has been initialised")
	} else {
		logger.Info("The service has already been initialised")
	}
	return nil
}

func (h *AdminStartService) buildClientAPI(ctx context.Context, logger *zap.Logger, cfg *network.Network, sessions *session.Sessions) (*jsonrpc.API, error) {
	clientAPILogger := logger.Named("client-api")

	nodeSelector, err := nodeapi.BuildRoundRobinSelectorWithRetryingNodes(clientAPILogger, cfg.API.GRPC.Hosts, cfg.API.GRPC.Retries)
	if err != nil {
		logger.Error("Could not build the node selector", zap.Error(err))
		return nil, err
	}

	interactor := h.interactorBuilderFunc(ctx)

	clientAPI, err := ClientAPI(clientAPILogger, h.walletStore, interactor, nodeSelector, sessions)
	if err != nil {
		logger.Error("Could not instantiate the client JSON-RPC API", zap.Error(err))
		return nil, fmt.Errorf("could not instantiate JSON-RPC API: %w", err)
	}

	return clientAPI, nil
}

func (h *AdminStartService) startService(srv *service.Service, logger *zap.Logger) error {
	logger.Info("Starting the HTTP server")
	if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("Error while running HTTP server", zap.Error(err))
		return err
	}
	return nil
}

func ensurePortCanBeBound(ctx context.Context, logger *zap.Logger, host string) *jsonrpc.ErrorDetails {
	url := "http://" + host
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, bytes.NewReader([]byte{}))
	if err != nil {
		logger.Error("Could not reach the service", zap.Error(err))
	}

	response, err := http.DefaultClient.Do(req)
	if err == nil {
		// If there is no error, it means the server managed to establish a
		// connection of some kind, whereas we would have liked it to be unable
		// to connect to anything, which would have implied this host is free to
		// use.
		logger.Error("Could not start the service as an application is already served on that url", zap.String("url", url))
		return servicePortAlreadyBound(fmt.Errorf("could not start the service as an application is already served on %q", url))
	}
	defer func() {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
	}()

	logger.Info("The URL seems free of use")
	return nil
}

func validateAdminStartServiceParams(rawParams jsonrpc.Params) (AdminStartServiceParams, error) {
	if rawParams == nil {
		return AdminStartServiceParams{}, ErrParamsRequired
	}

	params := AdminStartServiceParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminStartServiceParams{}, ErrParamsDoNotMatch
	}

	if params.Network == "" {
		return AdminStartServiceParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewAdminStartService(walletStore WalletStore, netStore NetworkStore, svcStore ServiceStore, policyBuilderFunc PolicyBuilderFunc, interactorBuilderFunc InteractorBuilderFunc, loggerBuilderFunc LoggerBuilderFunc, shutdownSwitchBuilderFunc ShutdownSwitchBuilder, servicesManager *ServicesManager) *AdminStartService {
	return &AdminStartService{
		walletStore:               walletStore,
		netStore:                  netStore,
		svcStore:                  svcStore,
		policyBuilderFunc:         policyBuilderFunc,
		interactorBuilderFunc:     interactorBuilderFunc,
		servicesManager:           servicesManager,
		loggerBuilderFunc:         loggerBuilderFunc,
		shutdownSwitchBuilderFunc: shutdownSwitchBuilderFunc,
	}
}
