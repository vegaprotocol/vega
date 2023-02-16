// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package coreapi

import (
	"context"
	"errors"
	"sync"

	lb "code.vegaprotocol.io/vega/libs/broker"

	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/coreapi/services"
	"code.vegaprotocol.io/vega/logging"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
)

var ErrServiceDisabled = errors.New("service disabled")

type Service struct {
	apipb.UnimplementedCoreStateServiceServer
	ctx context.Context
	cfg Config
	log *logging.Logger

	bmu    sync.RWMutex
	broker broker.Interface

	accounts     *services.Accounts
	assets       *services.Assets
	netparams    *services.NetParams
	netlimits    *services.NetLimits
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
	ctx context.Context, log *logging.Logger, cfg Config, broker broker.Interface,
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
	}

	if cfg.Assets {
		log.Info("starting assets core api")
		svc.assets = services.NewAssets(ctx)
	}

	if cfg.NetworkParameters {
		log.Info("starting network parameters core api")
		svc.netparams = services.NewNetParams(ctx)
	}

	if cfg.NetworkLimits {
		log.Info("starting network limits core api")
		svc.netlimits = services.NewNetLimits(ctx)
	}

	if cfg.Parties {
		log.Info("starting parties core api")
		svc.parties = services.NewParties(ctx)
	}

	if cfg.Validators {
		log.Info("starting validators core api")
		svc.validators = services.NewValidators(ctx)
	}

	if cfg.Markets {
		log.Info("starting markets core api")
		svc.markets = services.NewMarkets(ctx)
	}

	if cfg.Proposals {
		log.Info("starting proposals core api")
		svc.proposals = services.NewProposals(ctx)
	}

	if cfg.MarketsData {
		log.Info("starting marketsData core api")
		svc.marketsData = services.NewMarketsData(ctx)
	}

	if cfg.Votes {
		log.Info("starting votes core api")
		svc.votes = services.NewVotes(ctx)
	}

	if cfg.PartiesStake {
		log.Info("starting parties stake core api")
		svc.partiesStake = services.NewPartiesStake(ctx, log)
	}

	if cfg.Delegations {
		log.Info("starting delegations core api")
		svc.delegations = services.NewDelegations(ctx)
	}

	svc.subscribeAll()

	return svc
}

func (s *Service) UpdateBroker(broker broker.Interface) {
	s.bmu.Lock()
	defer s.bmu.Unlock()
	s.broker = broker
	s.subscribeAll()
}

func (s *Service) ListAccounts(
	ctx context.Context, in *apipb.ListAccountsRequest,
) (*apipb.ListAccountsResponse, error) {
	if !s.cfg.Accounts {
		return nil, ErrServiceDisabled
	}
	s.bmu.RLock()
	defer s.bmu.RUnlock()
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
	s.bmu.RLock()
	defer s.bmu.RUnlock()
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
	s.bmu.RLock()
	defer s.bmu.RUnlock()
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
	s.bmu.RLock()
	defer s.bmu.RUnlock()
	return &apipb.ListNetworkParametersResponse{
		NetworkParameters: s.netparams.List(in.NetworkParameterKey),
	}, nil
}

func (s *Service) ListNetworkLimits(
	ctx context.Context, in *apipb.ListNetworkLimitsRequest,
) (*apipb.ListNetworkLimitsResponse, error) {
	if !s.cfg.NetworkLimits {
		return nil, ErrServiceDisabled
	}
	s.bmu.RLock()
	defer s.bmu.RUnlock()
	return &apipb.ListNetworkLimitsResponse{
		NetworkLimits: s.netlimits.Get(),
	}, nil
}

func (s *Service) ListValidators(
	ctx context.Context, in *apipb.ListValidatorsRequest,
) (*apipb.ListValidatorsResponse, error) {
	if !s.cfg.Validators {
		return nil, ErrServiceDisabled
	}
	s.bmu.RLock()
	defer s.bmu.RUnlock()
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
	s.bmu.RLock()
	defer s.bmu.RUnlock()
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
	s.bmu.RLock()
	defer s.bmu.RUnlock()
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
	s.bmu.RLock()
	defer s.bmu.RUnlock()
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
	s.bmu.RLock()
	defer s.bmu.RUnlock()
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
	s.bmu.RLock()
	defer s.bmu.RUnlock()
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
	s.bmu.RLock()
	defer s.bmu.RUnlock()
	return &apipb.ListDelegationsResponse{
		Delegations: s.delegations.List(in.Party, in.Node, in.EpochSeq),
	}, nil
}

func (s *Service) subscribeAll() {
	subscribers := []lb.Subscriber{}

	if s.cfg.Accounts {
		subscribers = append(subscribers, s.accounts)
	}

	if s.cfg.Assets {
		subscribers = append(subscribers, s.assets)
	}

	if s.cfg.NetworkParameters {
		subscribers = append(subscribers, s.netparams)
	}

	if s.cfg.NetworkLimits {
		subscribers = append(subscribers, s.netlimits)
	}

	if s.cfg.Parties {
		subscribers = append(subscribers, s.parties)
	}

	if s.cfg.Validators {
		subscribers = append(subscribers, s.validators)
	}

	if s.cfg.Markets {
		subscribers = append(subscribers, s.markets)
	}

	if s.cfg.Proposals {
		subscribers = append(subscribers, s.proposals)
	}

	if s.cfg.MarketsData {
		subscribers = append(subscribers, s.marketsData)
	}

	if s.cfg.Votes {
		subscribers = append(subscribers, s.votes)
	}

	if s.cfg.PartiesStake {
		subscribers = append(subscribers, s.partiesStake)
	}

	if s.cfg.Delegations {
		subscribers = append(subscribers, s.delegations)
	}

	s.broker.SubscribeBatch(subscribers...)
}
