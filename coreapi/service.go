package coreapi

import (
	"context"
	"errors"

	apipb "code.vegaprotocol.io/protos/vega/api/v1"
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

	accounts     *services.Accounts
	assets       *services.Assets
	netparams    *services.NetParams
	parties      *services.Parties
	validators   *services.Validators
	markets      *services.Markets
	proposals    *services.Proposals
	votes        *services.Votes
	marketsData  *services.MarketsData
	partiesStake *services.PartiesStake
	delegations  *services.Delegations
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

	if cfg.Markets {
		log.Info("starting markets core api")
		svc.markets = services.NewMarkets(ctx)
		broker.SubscribeBatch(svc.markets)
	}

	if cfg.Proposals {
		log.Info("starting proposals core api")
		svc.proposals = services.NewProposals(ctx)
		broker.SubscribeBatch(svc.proposals)
	}

	if cfg.MarketsData {
		log.Info("starting marketsData core api")
		svc.marketsData = services.NewMarketsData(ctx)
		broker.SubscribeBatch(svc.marketsData)
	}

	if cfg.Votes {
		log.Info("starting votes core api")
		svc.votes = services.NewVotes(ctx)
		broker.SubscribeBatch(svc.votes)
	}

	if cfg.PartiesStake {
		log.Info("starting parties stake core api")
		svc.partiesStake = services.NewPartiesStake(ctx, log)
		broker.SubscribeBatch(svc.partiesStake)
	}

	if cfg.Delegations {
		log.Info("starting delegations core api")
		svc.delegations = services.NewDelegations(ctx)
		broker.SubscribeBatch(svc.delegations)
	}

	return svc
}

func (s *Service) ListAccounts(
	ctx context.Context, in *apipb.ListAccountsRequest,
) (*apipb.ListAccountsResponse, error) {
	if !s.cfg.Accounts {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListAccountsResponse{
		Accounts: s.accounts.List(in.Party, in.Market),
	}, nil
}

func (s *Service) ListAssets(
	ctx context.Context, in *apipb.ListAssetsRequest,
) (*apipb.ListAssetsResponse, error) {
	if !s.cfg.Assets {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListAssetsResponse{
		Assets: s.assets.List(in.Asset),
	}, nil
}

func (s *Service) ListParties(
	ctx context.Context, in *apipb.ListPartiesRequest,
) (*apipb.ListPartiesResponse, error) {
	if !s.cfg.Parties {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListPartiesResponse{
		Parties: s.parties.List(),
	}, nil
}

func (s *Service) ListNetworkParameters(
	ctx context.Context, in *apipb.ListNetworkParametersRequest,
) (*apipb.ListNetworkParametersResponse, error) {
	if !s.cfg.NetworkParameters {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListNetworkParametersResponse{
		NetworkParameters: s.netparams.List(in.NetworkParameterKey),
	}, nil
}

func (s *Service) ListValidators(
	ctx context.Context, in *apipb.ListValidatorsRequest,
) (*apipb.ListValidatorsResponse, error) {
	if !s.cfg.Validators {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListValidatorsResponse{
		Validators: s.validators.List(),
	}, nil
}

func (s *Service) ListMarkets(
	ctx context.Context, in *apipb.ListMarketsRequest,
) (*apipb.ListMarketsResponse, error) {
	if !s.cfg.Markets {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListMarketsResponse{
		Markets: s.markets.List(in.Market),
	}, nil
}

func (s *Service) ListProposals(
	ctx context.Context, in *apipb.ListProposalsRequest,
) (*apipb.ListProposalsResponse, error) {
	if !s.cfg.Proposals {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListProposalsResponse{
		Proposals: s.proposals.List(in.Proposal, in.Proposer),
	}, nil
}

func (s *Service) ListVotes(
	ctx context.Context, in *apipb.ListVotesRequest,
) (*apipb.ListVotesResponse, error) {
	if !s.cfg.Votes {
		return nil, ErrServiceDisabled
	}
	votes, err := s.votes.List(in.Proposal, in.Party)
	return &apipb.ListVotesResponse{
		Votes: votes,
	}, err
}

func (s *Service) ListMarketsData(
	ctx context.Context, in *apipb.ListMarketsDataRequest,
) (*apipb.ListMarketsDataResponse, error) {
	if !s.cfg.MarketsData {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListMarketsDataResponse{
		MarketsData: s.marketsData.List(in.Market),
	}, nil
}

func (s *Service) ListPartiesStake(
	ctx context.Context, in *apipb.ListPartiesStakeRequest,
) (*apipb.ListPartiesStakeResponse, error) {
	if !s.cfg.PartiesStake {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListPartiesStakeResponse{
		PartiesStake: s.partiesStake.List(in.Party),
	}, nil
}

func (s *Service) ListDelegations(
	ctx context.Context, in *apipb.ListDelegationsRequest,
) (*apipb.ListDelegationsResponse, error) {
	if !s.cfg.Delegations {
		return nil, ErrServiceDisabled
	}
	return &apipb.ListDelegationsResponse{
		Delegations: s.delegations.List(in.Party, in.Node, in.EpochSeq),
	}, nil
}
