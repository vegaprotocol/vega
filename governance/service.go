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
func (service *Svc) ReloadConf(cfg Config) {
	service.log.Info("reloading configuration")
	if service.log.GetLevel() != cfg.Level.Get() {
		service.log.Info("updating log level",
			logging.String("old", service.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		service.log.SetLevel(cfg.Level.Get())
	}

	service.mu.Lock()
	service.Config = cfg
	service.mu.Unlock()
}

// PrepareProposal performs basic validation and bundles together fields required for a proposal
func (service *Svc) PrepareProposal(
	ctx context.Context, party string, reference string, terms *types.Proposal_Terms,
) (*types.Proposal, error) {
	if !service.Config.Enabled {
		return nil, ErrGovernanceDisabled
	}
	if err := service.ValidateTerms(terms); err != nil {
		return nil, err
	}
	if !service.CanPropose(party) {
		return nil, ErrPartyCannotPropose
	}
	if len(reference) <= 0 {
		reference = fmt.Sprintf("proposal#%s", uuid.NewV4().String())
	}
	now, err := service.timeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	return &types.Proposal{
		Id:        "", // to be filled on submission
		Reference: reference,
		PartyID:   party,
		State:     types.Proposal_OPEN,
		Timestamp: now.Unix(),
		Terms:     terms,
		Votes:     nil,
	}, nil
}

// CanPropose checks if the party is allowed to submit new proposals
func (service *Svc) CanPropose(party string) bool {
	//TODO: read stake from somewhere
	return true
}

// ValidateTerms performs sanity checks:
// - network time restrictions parameters (voting duration, enactment date time);
// - network minimum participation requirement parameter.
func (service *Svc) ValidateTerms(terms *types.Proposal_Terms) error {
	if err := terms.Validate(); err != nil {
		return errors.Wrap(err, "proposal validation failed")
	}

	if terms.Parameters.MinParticipationStake < service.MinParticipationStake {
		return fmt.Errorf("minimum participation stake parameter must be at least %d",
			service.MinParticipationStake)
	}
	if terms.Parameters.CloseInDays < service.MinCloseInDays ||
		terms.Parameters.CloseInDays > service.MaxCloseInDays {
		return fmt.Errorf("close day must be between %d and %d",
			service.MinCloseInDays, service.MaxCloseInDays)
	}
	if terms.Parameters.EnactInDays < service.MinEnactInDays ||
		terms.Parameters.EnactInDays > service.MaxEnactInDays {
		return fmt.Errorf("enactment day must be between %d and %d",
			service.MinEnactInDays, service.MaxEnactInDays)
	}

	return nil
}
