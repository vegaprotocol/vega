package governance

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	// ErrInvalidProposalTerms is returned if basic validation has failed
	ErrInvalidProposalTerms = errors.New("invalid proposal terms")

	ErrMissingVoteData = errors.New("required fields from vote missing")
)

// Plugin ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/plugin_mock.go -package mocks code.vegaprotocol.io/vega/governance Plugin
type Plugin interface {
	Subscribe() (<-chan []types.GovernanceData, int64)
	Unsubscribe(int64)

	GetAllGovernanceData() []*types.GovernanceData
	GetProposalsInState(includeState types.Proposal_State) []*types.GovernanceData
	GetProposalsNotInState(excludeState types.Proposal_State) []*types.GovernanceData
	GetProposalsByMarket(marketID string) []*types.GovernanceData
	GetProposalsByParty(partyID string) []*types.GovernanceData

	GetProposalByID(id string) (*types.GovernanceData, error)
	GetProposalByReference(ref string) (*types.GovernanceData, error)
}

type networkParameters struct {
	minCloseInSeconds     int64
	maxCloseInSeconds     int64
	minEnactInSeconds     int64
	maxEnactInSeconds     int64
	minParticipationStake uint64
}

// Svc is governance service, responsible for managing proposals and votes.
type Svc struct {
	Config
	log    *logging.Logger
	mu     sync.Mutex
	plugin Plugin

	parameters networkParameters
}

// NewService creates new governance service instance
func NewService(log *logging.Logger, cfg Config, plugin Plugin) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	cfg.initParams() // ensures params are set

	return &Svc{
		Config: cfg,
		log:    log,
		plugin: plugin,
		parameters: networkParameters{
			minCloseInSeconds:     cfg.params.DefaultMinClose,
			maxCloseInSeconds:     cfg.params.DefaultMaxClose,
			minEnactInSeconds:     cfg.params.DefaultMinEnact,
			maxEnactInSeconds:     cfg.params.DefaultMaxEnact,
			minParticipationStake: cfg.params.DefaultMinParticipation,
		},
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
	cfg.params = s.Config.params
	s.Config = cfg
	s.mu.Unlock()
}

// ObserveGovernance - stream all governance updates to gRPC subscription stream
func (s *Svc) ObserveGovernance(ctx context.Context, retries int) <-chan []types.GovernanceData {
	var cfunc func()
	ctx, cfunc = context.WithCancel(ctx)
	// we're returning an extra channel because of the retry mechanic we want to add
	rCh := make(chan []types.GovernanceData)
	ch, chID := s.plugin.Subscribe()
	go func() {
		defer func() {
			// cancel context
			cfunc()
			// unsubscribe from plugin
			s.plugin.Unsubscribe(chID)
			// close channel to handler
			close(rCh)
		}()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug("proposal subscriber closed the connection")
				return
			case updates := <-ch:
				// received new proposal data
				retryCount := retries
				success := false
				for !success && retryCount >= 0 {
					select {
					case rCh <- updates:
						success = true
					default:
						s.log.Debug("failed to push proposal update onto subscriber channel")
						retryCount--
						time.Sleep(time.Millisecond * 10)
					}
				}
				if !success {
					s.log.Warn("Failed to push update to stream, reached end of retries")
					return
				}
			}
		}
	}()
	return rCh
}

// GetAllGovernanceData returns all governance data (proposals and votes)
func (s *Svc) GetAllGovernanceData() []*types.GovernanceData {
	return s.plugin.GetAllGovernanceData()
}

// GetProposalsInState returns proposals and their votes only including those in the `includeState`
func (s *Svc) GetProposalsInState(includeState types.Proposal_State) []*types.GovernanceData {
	return s.plugin.GetProposalsInState(includeState)
}

// GetProposalsNotInState returns proposals and their votes only excluding those in the `excludeState`
func (s *Svc) GetProposalsNotInState(excludeState types.Proposal_State) []*types.GovernanceData {
	return s.plugin.GetProposalsNotInState(excludeState)
}

// GetProposalsByMarket returns proposals and their votes by market that is affected by these proposals
func (s *Svc) GetProposalsByMarket(marketID string) []*types.GovernanceData {
	return s.plugin.GetProposalsByMarket(marketID)
}

// GetProposalsByParty returns proposals and their votes by party authoring them
func (s *Svc) GetProposalsByParty(partyID string) []*types.GovernanceData {
	return s.plugin.GetProposalsByParty(partyID)
}

// GetProposalByID returns a proposal and its votes by ID (if exists)
func (s *Svc) GetProposalByID(id string) (*types.GovernanceData, error) {
	return s.plugin.GetProposalByID(id)
}

// GetProposalByReference returns a proposal and its votes by reference (if exists)
func (s *Svc) GetProposalByReference(ref string) (*types.GovernanceData, error) {
	return s.plugin.GetProposalByReference(ref)
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
		State:     types.Proposal_OPEN,
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

// validateTerms performs sanity checks:
// - network time restrictions parameters (voting duration, enactment date time);
// - network minimum participation requirement parameter.
func (s *Svc) validateTerms(terms *types.ProposalTerms) error {
	if err := terms.Validate(); err != nil {
		return ErrInvalidProposalTerms
	}

	// we should be able to enact a proposal as soon as the voting is closed (and the proposal passed)
	if terms.EnactmentTimestamp < terms.ClosingTimestamp {
		return ErrInvalidProposalTerms
	}

	return nil
}
