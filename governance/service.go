package governance

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/types"

	uuid "github.com/satori/go.uuid"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/event_bus_mock.go -package mocks code.vegaprotocol.io/vega/governance EventBus
type EventBus interface {
	Subscribe(s broker.Subscriber) int
	Unsubscribe(id int)
}

// GovernanceDataSub - the subscriber that will be aggregating all governance data, used in non-stream calls
//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_data_sub_mock.go -package mocks code.vegaprotocol.io/vega/governance GovernanceDataSub
type GovernanceDataSub interface {
	Filter(uniqueVotes bool, filters ...subscribers.ProposalFilter) []*proto.GovernanceData
}

// VoteSub - subscriber containing all votes, which we can filter out
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vote_sub_mock.go -package mocks code.vegaprotocol.io/vega/governance VoteSub
type VoteSub interface {
	Filter(filters ...subscribers.VoteFilter) []*proto.Vote
}

// Svc is governance service, responsible for managing proposals and votes.
type Svc struct {
	Config
	log   *logging.Logger
	mu    sync.Mutex
	bus   EventBus
	gov   GovernanceDataSub
	votes VoteSub
	netp  NetParams
}

// NewService creates new governance service instance
func NewService(log *logging.Logger, cfg Config, bus EventBus, gov GovernanceDataSub, votes VoteSub, netp NetParams) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Svc{
		Config: cfg,
		log:    log,
		bus:    bus,
		gov:    gov,
		votes:  votes,
		netp:   netp,
	}
}

// ReloadConf updates the internal configuration of the collateral engine
func (s *Svc) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.mu.Lock()
	s.Config = cfg
	s.mu.Unlock()
}

// ObserveGovernance streams all governance updates
func (s *Svc) ObserveGovernance(ctx context.Context, retries int) <-chan []proto.GovernanceData {
	out := make(chan []proto.GovernanceData)
	ctx, cfunc := context.WithCancel(ctx)
	// use non-acking subscriber
	sub := subscribers.NewGovernanceSub(ctx, false)
	id := s.bus.Subscribe(sub)
	go func() {
		defer func() {
			s.bus.Unsubscribe(id)
			close(out)
			cfunc()
		}()
		ret := retries
		for {
			// wait for actual changes
			data := s.getCompleteGovernanceData(sub.GetGovernanceData())
			select {
			case <-ctx.Done():
				return
			case out <- data:
				ret = retries
			default:
				if ret == 0 {
					return
				}
				ret--
			}
		}
	}()
	return out
}

func (s *Svc) getCompleteGovernanceData(data []proto.GovernanceData) []proto.GovernanceData {
	gds := make([]proto.GovernanceData, 0, len(data))
	for _, gd := range data {
		var id string
		if gd.Proposal != nil && len(gd.Proposal.Id) > 0 {
			id = gd.Proposal.Id
		} else if len(gd.Yes) > 0 {
			id = gd.Yes[0].ProposalId
		} else if len(gd.No) > 0 {
			id = gd.No[0].ProposalId
		}
		if p, err := s.GetProposalByID(id); err != nil && p != nil {
			gds = append(gds, *p)
		} else {
			s.log.Debug("invalid proposal id",
				logging.String("proposal-id", id),
				logging.Error(err))
		}
	}

	return gds
}

// ObservePartyProposals streams proposals submitted by the specific party
func (s *Svc) ObservePartyProposals(ctx context.Context, retries int, partyID string) <-chan []proto.GovernanceData {
	ctx, cfunc := context.WithCancel(ctx)
	sub := subscribers.NewGovernanceSub(ctx, false, subscribers.Proposals(subscribers.ProposalByPartyID(partyID)))
	out := make(chan []proto.GovernanceData)
	id := s.bus.Subscribe(sub)
	go func() {
		defer func() {
			cfunc()
			s.bus.Unsubscribe(id)
			close(out)
		}()
		ret := retries
		for {
			data := s.getCompleteGovernanceData(sub.GetGovernanceData())
			select {
			case <-ctx.Done():
				return
			case out <- data:
				ret = retries
			default:
				if ret == 0 {
					return
				}
				ret--
			}
		}
	}()
	return out
}

// ObservePartyVotes streams votes cast by the specific party
func (s *Svc) ObservePartyVotes(ctx context.Context, retries int, partyID string) <-chan []proto.Vote {
	ctx, cfunc := context.WithCancel(ctx)
	out := make(chan []proto.Vote)
	// new subscriber, in "stream mode" (changes only), filtered by party ID
	// and make subscriber non-acking, missed votes are ignored
	sub := subscribers.NewVoteSub(ctx, true, false, subscribers.VoteByPartyID(partyID))
	id := s.bus.Subscribe(sub)
	go func() {
		defer func() {
			s.bus.Unsubscribe(id)
			close(out)
			cfunc()
		}()
		ret := retries
		for {
			data := sub.GetData()
			if len(data) == 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case out <- data:
				ret = retries
			default:
				if ret == 0 {
					return
				}
				ret--
			}
		}
	}()
	return out
}

// ObserveProposalVotes streams votes cast for/against specific proposal
func (s *Svc) ObserveProposalVotes(ctx context.Context, retries int, proposalID string) <-chan []proto.Vote {
	ctx, cfunc := context.WithCancel(ctx)
	out := make(chan []proto.Vote)
	// new subscriber, in "stream mode" (changes only), filtered by proposal ID
	sub := subscribers.NewVoteSub(ctx, true, false, subscribers.VoteByProposalID(proposalID))
	id := s.bus.Subscribe(sub)
	go func() {
		defer func() {
			s.bus.Unsubscribe(id)
			close(out)
			cfunc()
		}()
		ret := retries
		for {
			data := sub.GetData()
			if len(data) == 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case out <- data:
				ret = retries
			default:
				if ret == 0 {
					return
				}
				ret--
			}
		}
	}()
	return out
}

// GetProposals returns all governance data (proposals and votes)
func (s *Svc) GetProposals(inState *proto.Proposal_State) []*proto.GovernanceData {
	if inState != nil {
		return s.gov.Filter(true, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(true)
}

// GetProposalsByParty returns proposals and their votes by party authoring them
func (s *Svc) GetProposalsByParty(partyID string, inState *proto.Proposal_State) []*proto.GovernanceData {
	filters := []subscribers.ProposalFilter{
		subscribers.ProposalByPartyID(partyID),
	}
	if inState != nil {
		filters = append(filters, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(true, filters...)
}

// GetVotesByParty returns votes by party
func (s *Svc) GetVotesByParty(partyID string) []*proto.Vote {
	return s.votes.Filter(subscribers.VoteByPartyID(partyID))
}

// GetProposalByID returns a proposal and its votes by ID (if exists)
func (s *Svc) GetProposalByID(id string) (*proto.GovernanceData, error) {
	data := s.gov.Filter(true, subscribers.ProposalByID(id))
	if len(data) == 0 {
		return nil, ErrProposalNotFound
	}
	return data[0], nil
}

// GetProposalByReference returns a proposal and its votes by reference (if exists)
func (s *Svc) GetProposalByReference(ref string) (*proto.GovernanceData, error) {
	data := s.gov.Filter(true, subscribers.ProposalByReference(ref))
	if len(data) == 0 {
		return nil, ErrProposalNotFound
	}
	return data[0], nil
}

// GetNewMarketProposals returns proposals aiming to create new markets
func (s *Svc) GetNewMarketProposals(inState *proto.Proposal_State) []*proto.GovernanceData {
	filters := []subscribers.ProposalFilter{
		subscribers.ProposalByChange(subscribers.NewMarketProposal),
	}
	if inState != nil {
		filters = append(filters, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(true, filters...)
}

// GetUpdateMarketProposals returns proposals aiming to update existing markets
func (s *Svc) GetUpdateMarketProposals(marketID string, inState *proto.Proposal_State) []*proto.GovernanceData {
	filters := []subscribers.ProposalFilter{
		subscribers.ProposalByChange(subscribers.UpdateMarketProposal),
	}
	if inState != nil {
		filters = append(filters, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(true, filters...)
}

// GetNetworkParametersProposals returns proposals aiming to update network
func (s *Svc) GetNetworkParametersProposals(inState *proto.Proposal_State) []*proto.GovernanceData {
	filters := []subscribers.ProposalFilter{
		subscribers.ProposalByChange(subscribers.UpdateNetworkParameterProposal),
	}
	if inState != nil {
		filters = append(filters, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(
		true, // only latest votes,
		filters...,
	)
}

// GetNewAssetProposals returns proposals aiming to create new assets
func (s *Svc) GetNewAssetProposals(inState *proto.Proposal_State) []*proto.GovernanceData {
	filters := []subscribers.ProposalFilter{
		subscribers.ProposalByChange(subscribers.NewAssetPropopsal),
	}
	if inState != nil {
		filters = append(filters, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(true, filters...)
}

// PrepareProposal performs basic validation and bundles together fields required for a proposal
func (s *Svc) PrepareProposal(
	_ context.Context, reference string, terms *types.ProposalTerms,
) (*commandspb.ProposalSubmission, error) {
	if len(reference) <= 0 {
		reference = uuid.NewV4().String()
	}

	cmd := &commandspb.ProposalSubmission{
		Reference: reference,
		Terms:     terms.IntoProto(),
	}

	if err := commands.CheckProposalSubmission(cmd); err != nil {
		return nil, err
	}

	return cmd, nil
}

// PrepareVote - some additional validation on the vote message we're preparing
func (s *Svc) PrepareVote(cmd *types.VoteSubmission) (*commandspb.VoteSubmission, error) {
	vs := cmd.IntoProto()
	if err := commands.CheckVoteSubmission(vs); err != nil {
		return nil, err
	}

	return vs, nil
}
