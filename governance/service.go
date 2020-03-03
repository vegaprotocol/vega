package governance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	// ErrInvalidProposalTermsFmt is returned if basic validation has failed
	ErrInvalidProposalTermsFmt = errors.New("invalid proposal terms format")
)

const (
	defaultMinCloseInSeconds     = 2 * 24 * 60 * 60
	defaultMaxCloseInSeconds     = 365 * 24 * 60 * 60
	defaultMinEnactInSeconds     = 3 * 24 * 60 * 60
	defaultMaxEnactInSeconds     = 365 * 24 * 60 * 60
	defaultMinParticipationStake = 1
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

	return &Svc{
		Config:      cfg,
		log:         log,
		timeService: time,
		parameters: networkParameters{
			minCloseInSeconds:     defaultMinCloseInSeconds,
			maxCloseInSeconds:     defaultMaxCloseInSeconds,
			minEnactInSeconds:     defaultMinEnactInSeconds,
			maxEnactInSeconds:     defaultMaxEnactInSeconds,
			minParticipationStake: defaultMinParticipationStake,
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
		return errors.Wrap(err, "proposal validation failed")
	}

	if terms.MinParticipationStake < s.parameters.minParticipationStake {
		return fmt.Errorf("minimum participation stake parameter must be at least %d",
			s.parameters.minParticipationStake)
	}

	now, err := s.timeService.GetTimeNow()
	if err != nil {
		return errors.Wrap(err, "proposal validation failed")
	}

	minClose := now.Add(time.Duration(s.parameters.minCloseInSeconds) * time.Second)
	maxClose := now.Add(time.Duration(s.parameters.maxCloseInSeconds) * time.Second)

	if terms.ClosingTimestamp < minClose.UTC().Unix() ||
		terms.ClosingTimestamp > maxClose.UTC().Unix() {
		return fmt.Errorf("close day must be between %s and %s",
			vegatime.Format(minClose), vegatime.Format(maxClose))
	}

	minEnact := now.Add(time.Duration(s.parameters.minEnactInSeconds) * time.Second)
	maxEnact := now.Add(time.Duration(s.parameters.maxEnactInSeconds) * time.Second)
	if terms.EnactmentTimestamp < minEnact.UTC().Unix() ||
		terms.EnactmentTimestamp > maxEnact.UTC().Unix() {
		return fmt.Errorf("enactment day must be between %s and %s",
			vegatime.Format(minEnact), vegatime.Format(maxEnact))
	}

	return nil
}
