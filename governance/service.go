package governance

import (
	"context"
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
	// ErrInvalidProposalTerms is returned if basic validation has failed
	ErrInvalidProposalTerms = errors.New(invalidProposalTerms)

	ErrMissingVoteData = errors.New("required fields from vote missing")
)

// Plugin ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/plugin_mock.go -package mocks code.vegaprotocol.io/vega/governance Plugin
type Plugin interface {
	SubscribeAll() (<-chan []types.GovernanceData, int64)
	UnsubscribeAll(int64)

	SubscribePartyProposals(partyID string) (<-chan []types.GovernanceData, int64)
	UnsubscribePartyProposals(partyID string, idx int64)

	SubscribePartyVotes(partyID string) (<-chan []types.Vote, int64)
	UnsubscribePartyVotes(partyID string, idx int64)

	SubscribeProposalVotes(proposalID string) (<-chan []types.Vote, int64)
	UnsubscribeProposalVotes(proposalID string, idx int64)

	GetProposals(inState *types.Proposal_State) []*types.GovernanceData
	GetProposalsByParty(partyID string, inState *types.Proposal_State) []*types.GovernanceData
	GetVotesByParty(partyID string) []*types.Vote

	GetProposalByID(id string) (*types.GovernanceData, error)
	GetProposalByReference(ref string) (*types.GovernanceData, error)

	GetNewMarketProposals(inState *types.Proposal_State) []*types.GovernanceData
	GetUpdateMarketProposals(marketID string, inState *types.Proposal_State) []*types.GovernanceData
	GetNetworkParametersProposals(inState *types.Proposal_State) []*types.GovernanceData
	GetNewAssetProposals(inState *types.Proposal_State) []*types.GovernanceData
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/event_bus_mock.go -package mocks code.vegaprotocol.io/vega/governance EventBus
type EventBus interface {
	Subscribe(s broker.Subscriber, req bool) int
	Unsubscribe(id int)
}

// Svc is governance service, responsible for managing proposals and votes.
type Svc struct {
	Config
	log    *logging.Logger
	mu     sync.Mutex
	plugin Plugin
	bus    EventBus
}

// NewService creates new governance service instance
func NewService(log *logging.Logger, cfg Config, plugin Plugin, bus EventBus) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Svc{
		Config: cfg,
		log:    log,
		plugin: plugin,
		bus:    bus,
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

func streamVotes(ctx context.Context,
	retries int,
	input <-chan []types.Vote,
	output chan []types.Vote,
	log *logging.Logger,
) {
	for {
		select {
		case <-ctx.Done():
			log.Debug("votes subscriber closed the connection", logging.Error(ctx.Err()))
			return
		case updates := <-input:
			// received new data
			retryCount := retries
			success := false
			for !success && retryCount >= 0 {
				select {
				case output <- updates:
					success = true
				default:
					log.Debug("failed to push votes update onto subscriber channel")
					retryCount--
					time.Sleep(time.Millisecond * 10)
				}
			}
			if !success {
				log.Warn("Failed to push votes update to stream, reached end of retries")
				return
			}
		}
	}
}

// TODO: explore https://godoc.org/github.com/eapache/channels#Wrap to reduce copy-paste
func streamGovernance(ctx context.Context,
	retries int,
	input <-chan []types.GovernanceData,
	output chan []types.GovernanceData,
	log *logging.Logger,
) bool {

	select {
	case <-ctx.Done():
		log.Debug("governance subscriber closed the connection", logging.Error(ctx.Err()))
		return false
	case updates := <-input:
		// received new data
		retryCount := retries
		success := false
		for !success && retryCount >= 0 {
			select {
			case output <- updates:
				success = true
			default:
				log.Debug("failed to push governance update onto subscriber channel")
				retryCount--
				time.Sleep(time.Millisecond * 10)
			}
		}
		if !success {
			log.Warn("Failed to push governance update to stream, reached end of retries")
			return false
		}
	}
	return true
}

func (s *Svc) ObserveGovernanceSub(ctx context.Context, retries int) <-chan []types.GovernanceData {
	out := make(chan []types.GovernanceData)
	sub := subscribers.NewGovernanceSub(ctx)
	id := s.bus.Subscribe(sub, true)
	ctx, cfunc := context.WithCancel(ctx)
	go func() {
		defer func() {
			s.bus.Unsubscribe(id)
			close(out)
			cfunc()
		}()
		ret := retries
		for {
			data := sub.GetGovernanceData()
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

// ObserveGovernance streams all governance updates
func (s *Svc) ObserveGovernance(ctx context.Context, retries int) <-chan []types.GovernanceData {
	var cancelContext func()
	ctx, cancelContext = context.WithCancel(ctx)
	// we're returning an extra channel because of the retry mechanic we want to add
	output := make(chan []types.GovernanceData)
	input, inputIdx := s.plugin.SubscribeAll()

	go func() {
		defer func() {
			cancelContext()
			s.plugin.UnsubscribeAll(inputIdx)
			close(output)
		}()
		for streamGovernance(ctx, retries, input, output, s.log) {
		}
	}()
	return output
}

func (s *Svc) ObservePartyProposalsSub(ctx context.Context, retries int, partyID string) <-chan []types.GovernanceData {
	ctx, cfunc := context.WithCancel(ctx)
	sub := subscribers.NewGovernanceSub(ctx, subscribers.Proposals(subscribers.ProposalByPartyID(partyID)))
	out := make(chan []types.GovernanceData)
	id := s.bus.Subscribe(sub, true)
	go func() {
		defer func() {
			cfunc()
			s.bus.Unsubscribe(id)
			close(out)
		}()
		ret := retries
		for {
			data := sub.GetGovernanceData()
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

// ObservePartyProposals streams proposals submitted by the specific party
func (s *Svc) ObservePartyProposals(ctx context.Context, retries int, partyID string) <-chan []types.GovernanceData {
	var cancelContext func()
	ctx, cancelContext = context.WithCancel(ctx)
	output := make(chan []types.GovernanceData)
	input, inputIdx := s.plugin.SubscribePartyProposals(partyID)

	go func() {
		defer func() {
			cancelContext()
			s.plugin.UnsubscribePartyProposals(partyID, inputIdx)
			close(output)
		}()
		for streamGovernance(ctx, retries, input, output, s.log) {
		}
	}()
	return output
}

func (s *Svc) ObservePartyVotesSub(ctx context.Context, retries int, partyID string) <-chan []types.Vote {
	out := make(chan []types.Vote)
	// new subscriber, in "stream mode" (changes only), filtered by party ID
	sub := subscribers.NewVoteSub(ctx, true, subscribers.VoteByPartyID(partyID))
	id := s.bus.Subscribe(sub, true)
	ctx, cfunc := context.WithCancel(ctx)
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

// ObservePartyVotes streams votes cast by the specific party
func (s *Svc) ObservePartyVotes(ctx context.Context, retries int, partyID string) <-chan []types.Vote {
	var cancelContext func()
	ctx, cancelContext = context.WithCancel(ctx)
	output := make(chan []types.Vote)
	input, inputIdx := s.plugin.SubscribePartyVotes(partyID)

	go func() {
		defer func() {
			cancelContext()
			s.plugin.UnsubscribePartyVotes(partyID, inputIdx)
			close(output)
		}()
		streamVotes(ctx, retries, input, output, s.log)
	}()
	return output
}

func (s *Svc) ObserveProposalVotesSub(ctx context.Context, retries int, proposalID string) <-chan []types.Vote {
	out := make(chan []types.Vote)
	// new subscriber, in "stream mode" (changes only), filtered by proposal ID
	sub := subscribers.NewVoteSub(ctx, true, subscribers.VoteByProposalID(proposalID))
	id := s.bus.Subscribe(sub, true)
	ctx, cfunc := context.WithCancel(ctx)
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
	var cancelContext func()
	ctx, cancelContext = context.WithCancel(ctx)
	output := make(chan []types.Vote)
	input, inputIdx := s.plugin.SubscribeProposalVotes(proposalID)

	go func() {
		defer func() {
			cancelContext()
			s.plugin.UnsubscribeProposalVotes(proposalID, inputIdx)
			close(output)
		}()
		streamVotes(ctx, retries, input, output, s.log)
	}()
	return output
}

// GetProposals returns all governance data (proposals and votes)
func (s *Svc) GetProposals(inState *types.Proposal_State) []*types.GovernanceData {
	return s.plugin.GetProposals(inState)
}

// GetProposalsByParty returns proposals and their votes by party authoring them
func (s *Svc) GetProposalsByParty(partyID string, inState *types.Proposal_State) []*types.GovernanceData {
	return s.plugin.GetProposalsByParty(partyID, inState)
}

// GetVotesByParty returns votes by party
func (s *Svc) GetVotesByParty(partyID string) []*types.Vote {
	return s.plugin.GetVotesByParty(partyID)
}

// GetProposalByID returns a proposal and its votes by ID (if exists)
func (s *Svc) GetProposalByID(id string) (*types.GovernanceData, error) {
	return s.plugin.GetProposalByID(id)
}

// GetProposalByReference returns a proposal and its votes by reference (if exists)
func (s *Svc) GetProposalByReference(ref string) (*types.GovernanceData, error) {
	return s.plugin.GetProposalByReference(ref)
}

// GetNewMarketProposals returns proposals aiming to create new markets
func (s *Svc) GetNewMarketProposals(inState *types.Proposal_State) []*types.GovernanceData {
	return s.plugin.GetNewMarketProposals(inState)
}

// GetUpdateMarketProposals returns proposals aiming to update existing markets
func (s *Svc) GetUpdateMarketProposals(marketID string, inState *types.Proposal_State) []*types.GovernanceData {
	return s.plugin.GetUpdateMarketProposals(marketID, inState)
}

// GetNetworkParametersProposals returns proposals aiming to update network
func (s *Svc) GetNetworkParametersProposals(inState *types.Proposal_State) []*types.GovernanceData {
	return s.plugin.GetNetworkParametersProposals(inState)
}

// GetNewAssetProposals returns proposals aiming to create new assets
func (s *Svc) GetNewAssetProposals(inState *types.Proposal_State) []*types.GovernanceData {
	return s.plugin.GetNewAssetProposals(inState)
}

// PrepareProposal performs basic validation and bundles together fields required for a proposal
func (s *Svc) PrepareProposal(
	ctx context.Context, party string, reference string, terms *types.ProposalTerms,
) (*types.Proposal, error) {

	if err := s.validateTerms(terms); err != nil {
		return nil, err
	}
	if len(reference) <= 0 {
		reference = uuid.NewV4().String()
	}
	return &types.Proposal{
		Reference: reference,
		PartyID:   party,
		State:     types.Proposal_STATE_OPEN,
		Terms:     terms,
	}, nil
}

// PrepareVote - some additional validation on the vote message we're preparing
func (s *Svc) PrepareVote(vote *types.Vote) (*types.Vote, error) {
	// to check if the enum value is correct:
	_, ok := types.Vote_Value_value[vote.Value.String()]
	if vote.ProposalID == "" || vote.PartyID == "" || !ok {
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
		return ErrInvalidProposalTerms
	}

	return nil
}
