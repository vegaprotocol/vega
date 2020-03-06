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
)

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/governance TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
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
	log         *logging.Logger
	mu          sync.Mutex
	timeService TimeService

	parameters networkParameters
}

// NewService creates new governance service instance
func NewService(log *logging.Logger, cfg Config, time TimeService) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	cfg.initParams() // ensures params are set

	return &Svc{
		Config:      cfg,
		log:         log,
		timeService: time,
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
	s.Config = cfg
	s.mu.Unlock()
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

// validateTerms performs sanity checks:
// - network time restrictions parameters (voting duration, enactment date time);
// - network minimum participation requirement parameter.
func (s *Svc) validateTerms(terms *types.ProposalTerms) error {
	if err := terms.Validate(); err != nil {
		return ErrInvalidProposalTerms
	}

	now, err := s.timeService.GetTimeNow()
	if err != nil {
		return err
	}

	minClose := now.Add(time.Duration(s.parameters.minCloseInSeconds) * time.Second)
	// we can only check if the closing ts was in the past "too far in the future" might not apply
	// after the same proposal reaches the core (post consensus)
	if terms.ClosingTimestamp < minClose.UTC().Unix() {
		return ErrInvalidProposalTerms
	}

	// we should be able to enact a proposal as soon as the voting is closed (and the proposal passed)
	if terms.EnactmentTimestamp < terms.ClosingTimestamp {
		return ErrInvalidProposalTerms
	}

	return nil
}
