package processor

import (
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	ErrUnknownCommand                               = errors.New("unknown command when validating payload")
	ErrInvalidSignature                             = errors.New("invalid signature")
	ErrOrderSubmissionPartyAndPubKeyDoesNotMatch    = errors.New("order submission party and pubkey does not match")
	ErrOrderCancellationPartyAndPubKeyDoesNotMatch  = errors.New("order cancellation party and pubkey does not match")
	ErrOrderAmendmentPartyAndPubKeyDoesNotMatch     = errors.New("order amendment party and pubkey does not match")
	ErrProposalSubmissionPartyAndPubKeyDoesNotMatch = errors.New("proposal submission party and pubkey does not match")
	ErrVoteSubmissionPartyAndPubKeyDoesNotMatch     = errors.New("vote submission party and pubkey does not match")
	ErrWithdrawPartyAndPublKeyDoesNotMatch          = errors.New("withdraw party and pubkey does not match")
	ErrCommandKindUnknown                           = errors.New("unknown command kind when validating payload")
	ErrUnknownNodeKey                               = errors.New("node pubkey unknown")
	ErrUnknownProposal                              = errors.New("proposal unknown")
)

// ProcessorService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/processor_service_mock.go -package mocks code.vegaprotocol.io/vega/processor ProcessorService
type ProcessorService interface {
	SubmitOrder(order *types.Order) error
	CancelOrder(order *types.OrderCancellation) error
	AmendOrder(order *types.OrderAmendment) error
	NotifyTraderAccount(notify *types.NotifyTraderAccount) error
	Withdraw(*types.Withdraw) error
	SubmitProposal(proposal *types.Proposal) error
	VoteOnProposal(vote *types.Vote) error
}

type nodeProposal struct {
	*types.Proposal
	votes map[string]struct{}
}

// Processor handle processing of all transaction sent through the node
type Processor struct {
	log *logging.Logger
	Config
	svc           ProcessorService
	nodes         map[string]struct{} // all other nodes in the network
	nodeProposals map[string]*nodeProposal
}

// NewProcessor instantiates a new transactions processor
func New(log *logging.Logger, config Config, svc ProcessorService) *Processor {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Processor{
		log:           log,
		Config:        config,
		svc:           svc,
		nodes:         map[string]struct{}{},
		nodeProposals: map[string]*nodeProposal{},
	}
}

// ReloadConf update the internal configuration of the processor
func (p *Processor) ReloadConf(cfg Config) {
	p.log.Info("reloading configuration")
	if p.log.GetLevel() != cfg.Level.Get() {
		p.log.Info("updating log level",
			logging.String("old", p.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		p.log.SetLevel(cfg.Level.Get())
	}

	p.Config = cfg
}

func (p *Processor) getOrder(payload []byte) (*types.Order, error) {
	order := &types.Order{}
	err := proto.Unmarshal(payload, order)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (p *Processor) getOrderSubmission(payload []byte) (*types.Order, error) {
	orderSubmission := &types.OrderSubmission{}
	err := proto.Unmarshal(payload, orderSubmission)
	if err != nil {
		return nil, err
	}

	order := types.Order{
		Id:          orderSubmission.Id,
		MarketID:    orderSubmission.MarketID,
		PartyID:     orderSubmission.PartyID,
		Price:       orderSubmission.Price,
		Size:        orderSubmission.Size,
		Side:        orderSubmission.Side,
		TimeInForce: orderSubmission.TimeInForce,
		Type:        orderSubmission.Type,
		ExpiresAt:   orderSubmission.ExpiresAt,
		Reference:   orderSubmission.Reference,
		Status:      types.Order_Active,
		CreatedAt:   0,
		Remaining:   orderSubmission.Size,
	}

	return &order, nil
}

func (p *Processor) getOrderCancellation(payload []byte) (*types.OrderCancellation, error) {
	order := &types.OrderCancellation{}
	err := proto.Unmarshal(payload, order)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (p *Processor) getOrderAmendment(payload []byte) (*types.OrderAmendment, error) {
	amendment := &types.OrderAmendment{}
	err := proto.Unmarshal(payload, amendment)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding order to proto")
	}
	return amendment, nil
}

func (p *Processor) getNotifyTraderAccount(payload []byte) (*types.NotifyTraderAccount, error) {
	notif := &types.NotifyTraderAccount{}
	err := proto.Unmarshal(payload, notif)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding order to proto")
	}
	return notif, nil
}

func (p *Processor) getWithdraw(payload []byte) (*types.Withdraw, error) {
	w := &types.Withdraw{}
	err := proto.Unmarshal(payload, w)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding order to proto")
	}
	return w, nil
}

func (p *Processor) getProposalSubmission(payload []byte) (*types.Proposal, error) {
	proposalSubmission := &types.Proposal{}
	err := proto.Unmarshal(payload, proposalSubmission)
	if err != nil {
		return nil, err
	}
	return proposalSubmission, nil
}

func (p *Processor) getVoteSubmission(payload []byte) (*types.Vote, error) {
	voteSubmission := &types.Vote{}
	err := proto.Unmarshal(payload, voteSubmission)
	if err != nil {
		return nil, err
	}
	return voteSubmission, nil
}

// ValidateSigned - validates a signed transaction. This sits here because it's actual data processing
// related. We need to unmarshal the payload to validate the partyID
func (p *Processor) ValidateSigned(key, data []byte, cmd blockchain.Command) error {
	switch cmd {
	case blockchain.SubmitOrderCommand:
		order, err := p.getOrderSubmission(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if order.PartyID != hex.EncodeToString(key) {
			return ErrOrderSubmissionPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.CancelOrderCommand:
		order, err := p.getOrderCancellation(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if order.PartyID != hex.EncodeToString(key) {
			return ErrOrderCancellationPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.AmendOrderCommand:
		order, err := p.getOrderAmendment(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if order.PartyID != hex.EncodeToString(key) {
			return ErrOrderAmendmentPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.ProposeCommand:
		proposal, err := p.getProposalSubmission(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if proposal.PartyID != hex.EncodeToString(key) {
			return ErrProposalSubmissionPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.VoteCommand:
		vote, err := p.getVoteSubmission(data)
		if err != nil {
			return err
		}
		// partyID is hex encoded pubkey
		if vote.PartyID != hex.EncodeToString(key) {
			return ErrVoteSubmissionPartyAndPubKeyDoesNotMatch
		}
		return nil
	case blockchain.WithdrawCommand:
		withdraw, err := p.getWithdraw(data)
		if err != nil {
			return err
		}
		if withdraw.PartyID != hex.EncodeToString(key) {
			return ErrWithdrawPartyAndPublKeyDoesNotMatch
		}
		return nil
	}
	return errors.New("unknown command when validating payload")
}

// Process performs validation and then sends the command and data to
// the underlying blockchain service handlers e.g. submit order, etc.
func (p *Processor) Process(data []byte, cmd blockchain.Command) (err error) {
	// first is that a signed or unsigned command?
	switch cmd {
	case blockchain.SubmitOrderCommand:
		order, err := p.getOrderSubmission(data)
		if err != nil {
			return err
		}
		err = p.svc.SubmitOrder(order)
	case blockchain.CancelOrderCommand:
		order, err := p.getOrderCancellation(data)
		if err != nil {
			return err
		}
		err = p.svc.CancelOrder(order)
	case blockchain.AmendOrderCommand:
		order, err := p.getOrderAmendment(data)
		if err != nil {
			return err
		}
		err = p.svc.AmendOrder(order)
	case blockchain.WithdrawCommand:
		withdraw, err := p.getWithdraw(data)
		if err != nil {
			return err
		}
		err = p.svc.Withdraw(withdraw)
	case blockchain.ProposeCommand:
		proposal, err := p.getProposalSubmission(data)
		if err != nil {
			return err
		}
		// proposal is a new asset proposal?
		if na := proposal.Terms.GetNewAsset(); na != nil {
			p.nodeProposals[proposal.Reference] = &nodeProposal{
				Proposal: proposal,
				votes:    map[string]struct{}{},
			}
			// @TODO validate proposal here + cast vote
			return nil
		}
		err = p.svc.SubmitProposal(proposal)
	case blockchain.VoteCommand:
		vote, err := p.getVoteSubmission(data)
		if err != nil {
			return err
		}
		err = p.svc.VoteOnProposal(vote)
	case blockchain.RegisterNodeCommand:
		node, err := p.getNodeRegistration(data)
		if err != nil {
			return err
		}
		p.nodes[node.PubKey] = struct{}{}
	case blockchain.NodeVoteCommand:
		vote, err := p.getNodeVote(data)
		if err != nil {
			return err
		}
		if _, ok := p.nodes[vote.PubKey]; !ok {
			return ErrUnknownNodeKey
		}
		prop, ok := p.nodeProposals[vote.Reference]
		if !ok {
			return ErrUnknownProposal
		}
		prop.votes[vote.PubKey] = struct{}{}
	case blockchain.NotifyTraderAccountCommand:
		notify, err := p.getNotifyTraderAccount(data)
		if err != nil {
			return err
		}
		return p.svc.NotifyTraderAccount(notify)
	default:
		p.log.Warn("Unknown command received", logging.String("command", cmd.String()))
		err = fmt.Errorf("unknown command received: %s", cmd)
	}
	return err
}

func (p *Processor) getNodeVote(payload []byte) (*types.NodeVote, error) {
	vote := &types.NodeVote{}
	if err := proto.Unmarshal(payload, vote); err != nil {
		return nil, err
	}
	return vote, nil
}

func (p *Processor) getNodeRegistration(payload []byte) (*types.NodeRegistration, error) {
	cmd := &types.NodeRegistration{}
	err := proto.Unmarshal(payload, cmd)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}
