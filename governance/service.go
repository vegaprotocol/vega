package governance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const invalidProposalTerms = "invalid proposal terms"

var (
	ErrMissingVoteData          = errors.New("required fields from vote missing")
	ErrUnsupportedProposalTerms = errors.New("unsupported proposal terms")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/event_bus_mock.go -package mocks code.vegaprotocol.io/vega/governance EventBus
type EventBus interface {
	Subscribe(s broker.Subscriber) int
	Unsubscribe(id int)
}

// GovernanceDataSub - the subscriber that will be aggregating all governance data, used in non-stream calls
//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_data_sub_mock.go -package mocks code.vegaprotocol.io/vega/governance GovernanceDataSub
type GovernanceDataSub interface {
	Filter(uniqueVotes bool, filters ...subscribers.ProposalFilter) []*types.GovernanceData
}

// VoteSub - subscriber containing all votes, which we can filter out
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vote_sub_mock.go -package mocks code.vegaprotocol.io/vega/governance VoteSub
type VoteSub interface {
	Filter(filters ...subscribers.VoteFilter) []*types.Vote
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
func (s *Svc) ObserveGovernance(ctx context.Context, retries int) <-chan []types.GovernanceData {
	out := make(chan []types.GovernanceData)
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

func (s *Svc) getCompleteGovernanceData(data []types.GovernanceData) []types.GovernanceData {
	gds := make([]types.GovernanceData, 0, len(data))
	for _, gd := range data {
		var id string
		if gd.Proposal != nil && len(gd.Proposal.Id) > 0 {
			id = gd.Proposal.Id
		} else if len(gd.Yes) > 0 {
			id = gd.Yes[0].ProposalId
		} else if len(gd.No) > 0 {
			id = gd.No[0].ProposalId
		}
		p, _ := s.GetProposalByID(id)
		gds = append(gds, *p)
	}

	return gds
}

// ObservePartyProposals streams proposals submitted by the specific party
func (s *Svc) ObservePartyProposals(ctx context.Context, retries int, partyID string) <-chan []types.GovernanceData {
	ctx, cfunc := context.WithCancel(ctx)
	sub := subscribers.NewGovernanceSub(ctx, false, subscribers.Proposals(subscribers.ProposalByPartyID(partyID)))
	out := make(chan []types.GovernanceData)
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
func (s *Svc) ObservePartyVotes(ctx context.Context, retries int, partyID string) <-chan []types.Vote {
	ctx, cfunc := context.WithCancel(ctx)
	out := make(chan []types.Vote)
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
func (s *Svc) ObserveProposalVotes(ctx context.Context, retries int, proposalID string) <-chan []types.Vote {
	ctx, cfunc := context.WithCancel(ctx)
	out := make(chan []types.Vote)
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
func (s *Svc) GetProposals(inState *types.Proposal_State) []*types.GovernanceData {
	if inState != nil {
		return s.gov.Filter(true, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(true)
}

// GetProposalsByParty returns proposals and their votes by party authoring them
func (s *Svc) GetProposalsByParty(partyID string, inState *types.Proposal_State) []*types.GovernanceData {
	filters := []subscribers.ProposalFilter{
		subscribers.ProposalByPartyID(partyID),
	}
	if inState != nil {
		filters = append(filters, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(true, filters...)
}

// GetVotesByParty returns votes by party
func (s *Svc) GetVotesByParty(partyID string) []*types.Vote {
	return s.votes.Filter(subscribers.VoteByPartyID(partyID))
}

// GetProposalByID returns a proposal and its votes by ID (if exists)
func (s *Svc) GetProposalByID(id string) (*types.GovernanceData, error) {
	data := s.gov.Filter(true, subscribers.ProposalByID(id))
	if len(data) == 0 {
		return nil, ErrProposalNotFound
	}
	return data[0], nil
}

// GetProposalByReference returns a proposal and its votes by reference (if exists)
func (s *Svc) GetProposalByReference(ref string) (*types.GovernanceData, error) {
	data := s.gov.Filter(true, subscribers.ProposalByReference(ref))
	if len(data) == 0 {
		return nil, ErrProposalNotFound
	}
	return data[0], nil
}

// GetNewMarketProposals returns proposals aiming to create new markets
func (s *Svc) GetNewMarketProposals(inState *types.Proposal_State) []*types.GovernanceData {
	filters := []subscribers.ProposalFilter{
		subscribers.ProposalByChange(subscribers.NewMarketProposal),
	}
	if inState != nil {
		filters = append(filters, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(true, filters...)
}

// GetUpdateMarketProposals returns proposals aiming to update existing markets
func (s *Svc) GetUpdateMarketProposals(marketID string, inState *types.Proposal_State) []*types.GovernanceData {
	filters := []subscribers.ProposalFilter{
		subscribers.ProposalByChange(subscribers.UpdateMarketProposal),
	}
	if inState != nil {
		filters = append(filters, subscribers.ProposalByState(*inState))
	}
	return s.gov.Filter(true, filters...)
}

// GetNetworkParametersProposals returns proposals aiming to update network
func (s *Svc) GetNetworkParametersProposals(inState *types.Proposal_State) []*types.GovernanceData {
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
func (s *Svc) GetNewAssetProposals(inState *types.Proposal_State) []*types.GovernanceData {
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
	ctx context.Context, reference string, terms *types.ProposalTerms,
) (*types.ProposalSubmission, error) {
	if err := s.validateTerms(terms); err != nil {
		return nil, err
	}
	if len(reference) <= 0 {
		reference = uuid.NewV4().String()
	}
	return &types.ProposalSubmission{
		Reference: reference,
		Terms:     terms,
	}, nil
}

// PrepareVote - some additional validation on the vote message we're preparing
func (s *Svc) PrepareVote(vote *types.VoteSubmission) (*types.VoteSubmission, error) {
	// to check if the enum value is correct:
	_, ok := types.Vote_Value_value[vote.Value.String()]
	if vote.ProposalId == "" || !ok {
		return nil, ErrMissingVoteData
	}
	return vote, nil
}

// validateTerms performs trivial sanity check
func (s *Svc) validateTerms(terms *types.ProposalTerms) error {
	if err := terms.Validate(); err != nil {
		return errors.Wrap(err, invalidProposalTerms)
	}

	// we should be able to enact a proposal as soon as the voting is closed (and the proposal passed)
	if terms.EnactmentTimestamp < terms.ClosingTimestamp {
		enactTime := time.Unix(terms.EnactmentTimestamp, 0)
		closeTime := time.Unix(terms.ClosingTimestamp, 0)
		return fmt.Errorf("proposal enactment time cannot be before closing time, expected > %v got %v", closeTime, enactTime)
	}

	if terms.ValidationTimestamp > 0 && terms.ValidationTimestamp >= terms.ClosingTimestamp {
		validationTime := time.Unix(terms.ValidationTimestamp, 0)
		closeTime := time.Unix(terms.ClosingTimestamp, 0)
		return fmt.Errorf("proposal closing time cannot be before validation time, expected > %v got %v", validationTime, closeTime)
	}

	return s.validateProposalChanges(terms)
}

func (s *Svc) validateProposalChanges(terms *types.ProposalTerms) error {
	switch c := terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		return s.validateNewMarketChanges(terms, c.NewMarket)
	case *types.ProposalTerms_UpdateNetworkParameter:
		return s.validateUpdateNetworkParameterChanges(c.UpdateNetworkParameter)
	case *types.ProposalTerms_NewAsset:
		return s.validateNewAssetChanges(c.NewAsset)
	default:
		return ErrUnsupportedProposalTerms
	}
}

func (s *Svc) validateUpdateNetworkParameterChanges(np *types.UpdateNetworkParameter) (err error) {
	_, err = validateNetworkParameterUpdate(s.netp, np.Changes)
	return
}

func (s *Svc) validateNewAssetChanges(np *types.NewAsset) (err error) {
	_, err = validateNewAsset(np.Changes)
	return
}

func (s *Svc) validateNewMarketChanges(
	terms *types.ProposalTerms, nm *types.NewMarket) (err error) {
	closeTime := time.Unix(terms.ClosingTimestamp, 0)
	enactTime := time.Unix(terms.EnactmentTimestamp, 0)

	// just validate things which cannot be done straight with
	_, err = validateNewMarket(
		time.Time{}, nm, nil, false, s.netp, enactTime.Sub(closeTime))
	return
}
