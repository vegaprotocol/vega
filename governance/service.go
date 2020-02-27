package governance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	// ErrInvalidProposalTermsFmt is returned if basic validation has failed
	ErrInvalidProposalTermsFmt = errors.New("invalid proposal terms format")
)

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/governance TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
}

// Svc is governance service, responsible for managing proposals and votes.
type Svc struct {
	Config
	log              *logging.Logger
	mu               sync.Mutex
	timeService      TimeService
	referenceCounter uint64
}

// NewService creates new governance service instance
func NewService(log *logging.Logger, cfg Config, time TimeService) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Svc{
		Config:           cfg,
		log:              log,
		timeService:      time,
		referenceCounter: 0,
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
	ctx context.Context, author string, reference string, terms *types.Proposal_Terms,
) (*types.Proposal, error) {
	if err := service.validateProposal(terms); err != nil {
		return nil, err
	}
	if len(reference) <= 0 {
		service.referenceCounter++
		reference = fmt.Sprintf("proposal#%d", service.referenceCounter)
	}
	now, err := service.timeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	return &types.Proposal{
		Id:        "", // to be filled on submission
		Reference: reference,
		Author:    author,
		State:     types.Proposal_OPEN,
		Timestamp: now.Unix(),
		Terms:     terms,
		Votes:     nil,
	}, nil
}

// validateProposal performs basic consistency checks:
// - user ability to submit new proposals;
// - network time restrictions parameters (voting duration, enactment date time);
// - network minimum participation requirement parameter.
func (service *Svc) validateProposal(proposal *types.Proposal_Terms) error {
	//TODO: check if proposal.Author is valid (not just empty)
	//TODO: proposal.Parameters have to be checked against network parameters
	if err := proposal.Validate(); err != nil {
		return errors.Wrap(err, "order validation failed")
	}
	return nil
}
