package governance

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"
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

// Blockchain ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_mock.go -package mocks code.vegaprotocol.io/vega/governance  Blockchain
type Blockchain interface {
	SubmitTransaction(ctx context.Context, raw []byte) (bool, error)
}

// Svc is governance service, responsible for managing proposals and votes.
type Svc struct {
	Config
	log *logging.Logger

	mu sync.Mutex

	timeService TimeService
	blockchain  Blockchain
}

// NewService creates new governance service instance
func NewService(log *logging.Logger, cfg Config, time TimeService, client Blockchain) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Svc{
		Config: cfg,
		log:    log,

		timeService: time,
		blockchain:  client,
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

// submitProposal validates a proposal and submits it to the chain if valid
func (service *Svc) submitProposal(author string, proposal *types.Proposal_Terms) (*types.Proposal, error) {
	if err := service.validateProposal(proposal); err != nil {
		return nil, err
	}

	return &types.Proposal{
		Id:        "", ///< generate id, perhaps by simply last known proposal + 1
		State:     types.Proposal_OPEN,
		Author:    author,
		Timestamp: time.Now().Unix(),
		Proposal:  proposal,
		Votes:     nil, ///< submitter's stake
	}, nil
}

// Propose allows submitting new proposals
//TODO: this should probably go into a separate type in api package
func (service *Svc) Propose(ctx context.Context, proposalRequest *api.SubmitProposalRequest) (*api.SubmitProposalResponse, error) {
	author := "xxxxxxxxxxx" ///< derive from proposalRequest.Token

	proposal, err := service.submitProposal(author, proposalRequest.Submission)
	if err != nil {
		return nil, err
	}
	return &api.SubmitProposalResponse{
		Proposal: proposal,
	}, nil
}
