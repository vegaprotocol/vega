package coreapi

import (
	"context"
	"errors"

	coreapipb "code.vegaprotocol.io/protos/vega/coreapi/v1"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/coreapi/services"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrServiceDisabled = errors.New("service disabled")
)

type Service struct {
	ctx    context.Context
	broker broker.BrokerI
	cfg    Config
	log    *logging.Logger

	accounts   *services.Accounts
	assets     *services.Assets
	netparams  *services.NetParams
	parties    *services.Parties
	validators *services.Validators
}

func NewService(
	ctx context.Context, log *logging.Logger, cfg Config, broker broker.BrokerI,
) *Service {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.LogLevel.Get())
	svc := &Service{
		broker: broker,
		cfg:    cfg,
		ctx:    ctx,
		log:    log,
	}

	if cfg.Accounts {
		log.Info("starting accounts core api")
		svc.accounts = services.NewAccounts(ctx)
		broker.SubscribeBatch(svc.accounts)
	}

	if cfg.Assets {
		log.Info("starting assets core api")
		svc.assets = services.NewAssets(ctx)
		broker.SubscribeBatch(svc.assets)
	}

	if cfg.NetworkParameters {
		log.Info("starting network parameters core api")
		svc.netparams = services.NewNetParams(ctx)
		broker.SubscribeBatch(svc.netparams)
	}

	if cfg.Parties {
		log.Info("starting parties core api")
		svc.parties = services.NewParties(ctx)
		broker.SubscribeBatch(svc.parties)
	}

	if cfg.Validators {
		log.Info("starting validators core api")
		svc.validators = services.NewValidators(ctx)
		broker.SubscribeBatch(svc.validators)
	}

	return svc
}

func (s *Service) ListAccounts(
	ctx context.Context, in *coreapipb.ListAccountsRequest,
) (*coreapipb.ListAccountsResponse, error) {
	if !s.cfg.Accounts {
		return nil, ErrServiceDisabled
	}
	return &coreapipb.ListAccountsResponse{
		Accounts: s.accounts.List(in.Party, in.Market),
	}, nil
}

func (s *Service) ListAssets(
	ctx context.Context, in *coreapipb.ListAssetsRequest,
) (*coreapipb.ListAssetsResponse, error) {
	if !s.cfg.Assets {
		return nil, ErrServiceDisabled
	}
	return &coreapipb.ListAssetsResponse{
		Assets: s.assets.List(in.Asset),
	}, nil
}

func (s *Service) ListParties(
	ctx context.Context, in *coreapipb.ListPartiesRequest,
) (*coreapipb.ListPartiesResponse, error) {
	if !s.cfg.Parties {
		return nil, ErrServiceDisabled
	}
	return &coreapipb.ListPartiesResponse{
		Parties: s.parties.List(),
	}, nil
}

func (s *Service) ListNetworkParameters(
	ctx context.Context, in *coreapipb.ListNetworkParametersRequest,
) (*coreapipb.ListNetworkParametersResponse, error) {
	if !s.cfg.NetworkParameters {
		return nil, ErrServiceDisabled
	}
	return &coreapipb.ListNetworkParametersResponse{
		NetworkParameters: s.netparams.List(in.NetworkParameterKey),
	}, nil
}

func (s *Service) ListValidators(
	ctx context.Context, in *coreapipb.ListValidatorsRequest,
) (*coreapipb.ListValidatorsResponse, error) {
	if !s.cfg.Validators {
		return nil, ErrServiceDisabled
	}
	return &coreapipb.ListValidatorsResponse{
		Validators: s.validators.List(),
	}, nil
}
