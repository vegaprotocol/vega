package governance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	// ErrInvalidProposalTermsFmt is returned if basic validation has failed
	ErrInvalidProposalTermsFmt = errors.New("invalid proposal terms format")
	// ErrPartyCannotPropose is returned when proposing party does not have sufficient stake
	ErrPartyCannotPropose = errors.New("party cannot submit new proposals")
	// ErrGovernanceDisabled is returned if governance API was used when disabled
	ErrGovernanceDisabled = errors.New("governance API has been disabled")
)

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/governance TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
}

type networkParameters struct {
	minCloseInDays        uint64
	maxCloseInDays        uint64
	minEnactInDays        uint64
	maxEnactInDays        uint64
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

	return &Svc{
		Config:      cfg,
		log:         log,
		timeService: time,
		parameters: networkParameters{
			minCloseInDays:        cfg.MinCloseInDays,
			maxCloseInDays:        cfg.MaxCloseInDays,
			minEnactInDays:        cfg.MinEnactInDays,
			maxEnactInDays:        cfg.MaxEnactInDays,
			minParticipationStake: cfg.MinParticipationStake,
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
	ctx context.Context, party string, reference string, terms *types.Proposal_Terms,
) (*types.Proposal, error) {
	if !s.Config.Enabled {
		return nil, ErrGovernanceDisabled
	}
	if err := s.ValidateTerms(terms); err != nil {
		return nil, err
	}
	if len(reference) <= 0 {
		reference = fmt.Sprintf("proposal#%s", uuid.NewV4().String())
	}
	now, err := s.timeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	return &types.Proposal{
		Id:        "", // to be filled after submission
		Reference: reference,
		PartyID:   party,
		State:     types.Proposal_OPEN,
		Timestamp: now.Unix(),
		Terms:     terms,
		Votes:     nil,
	}, nil
}

// ValidateTerms performs sanity checks:
// - network time restrictions parameters (voting duration, enactment date time);
// - network minimum participation requirement parameter.
func (s *Svc) ValidateTerms(terms *types.Proposal_Terms) error {
	if err := terms.Validate(); err != nil {
		return errors.Wrap(err, "proposal validation failed")
	}

	if terms.Parameters.MinParticipationStake < s.MinParticipationStake {
		return fmt.Errorf("minimum participation stake parameter must be at least %d",
			s.MinParticipationStake)
	}
	if terms.Parameters.CloseInDays < s.MinCloseInDays ||
		terms.Parameters.CloseInDays > s.MaxCloseInDays {
		return fmt.Errorf("close day must be between %d and %d",
			s.MinCloseInDays, s.MaxCloseInDays)
	}
	if terms.Parameters.EnactInDays < s.MinEnactInDays ||
		terms.Parameters.EnactInDays > s.MaxEnactInDays {
		return fmt.Errorf("enactment day must be between %d and %d",
			s.MinEnactInDays, s.MaxEnactInDays)
	}

	return nil
}
