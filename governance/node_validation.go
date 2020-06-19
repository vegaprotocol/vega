package governance

import (
	"context"
	"encoding/hex"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrNoNodeValidationRequired            = errors.New("no node validation required")
	ErrProposalReferenceDuplicate          = errors.New("proposal duplicate")
	ErrProposalValidationTimestampInvalid  = errors.New("proposal validation timestamp invalid")
	ErrInvalidProposalReferenceForNodeVote = errors.New("invalid reference proposal for node vote")
	ErrDuplicateVoteFromNode               = errors.New("duplicate vote from node")
	ErrNodeIsNotAValidator                 = errors.New("node is not a validator")
	ErrVegaWalletRequired                  = errors.New("vega wallet required")
)

const (
	minValidationPeriod = 600       // ten minutes
	maxValidationPeriod = 48 * 3600 // 2 days
	nodeApproval        = 1         // float for percentage
)

const (
	notValidatedProposal uint32 = iota
	validatedProposal
	voteSentProposal
)

type NodeValidation struct {
	log *logging.Logger
	// used for nodes proposals
	top           ValidatorTopology
	nodeProposals map[string]*nodeProposal
	assets        Assets
	cmd           Commander
	vegaWallet    nodewallet.Wallet

	currentTimestamp time.Time
	isValidator      bool
}

type nodeProposal struct {
	*types.Proposal
	votes     map[string]struct{}
	validTime time.Time
	// use for the node internal validation
	validState uint32
	cancel     func()
}

func NewNodeValidation(
	log *logging.Logger,
	top ValidatorTopology,
	wallet Wallet,
	cmd Commander,
	assets Assets,
	now time.Time,
	isValidator bool,
) (*NodeValidation, error) {

	vegaWallet, ok := wallet.Get(nodewallet.Vega)
	if !ok {
		return nil, ErrVegaWalletRequired
	}

	return &NodeValidation{
		log:              log,
		top:              top,
		nodeProposals:    map[string]*nodeProposal{},
		assets:           assets,
		cmd:              cmd,
		vegaWallet:       vegaWallet,
		currentTimestamp: now,
		isValidator:      isValidator,
	}, nil
}

// returns validated proposal by all nodes
func (n *NodeValidation) OnChainTimeUpdate(t time.Time) []*types.Proposal {
	n.currentTimestamp = t

	accepted := []*types.Proposal{}

	// check that any proposal is ready
	for k, prop := range n.nodeProposals {
		// this proposal has passed the node-voting period, or all nodes have voted/approved
		// time expired, or all vote agregated, and own vote sent
		state := atomic.LoadUint32(&prop.validState)
		if prop.validTime.Before(t) || (len(prop.votes) == n.top.Len() && state == voteSentProposal) {
			// if not all nodes have approved, just remove
			if len(prop.votes) < n.top.Len() {
				n.log.Warn("proposal was not accepted by all nodes",
					logging.String("proposal", prop.Proposal.String()),
					logging.Int("vote-count", len(prop.votes)),
					logging.Int("node-count", n.top.Len()),
				)
			} else {
				// proposal was accepted by all nodes, returns it to the governance engine
				accepted = append(accepted, prop.Proposal)
			}

			// either proposal wasn't accepted, or it's been passed on to governance
			delete(n.nodeProposals, k)
			// cancelling this but it should already be exited if th proposal
			// was valid
			prop.cancel()
		}

		// or check if the proposal if valid,
		// if it is, we will send our own message through the network.
		if state == validatedProposal {
			// if not a validator no need to send the vote
			if n.isValidator {
				nv := &types.NodeVote{
					PubKey:    n.vegaWallet.PubKeyOrAddress(),
					Reference: prop.Reference,
				}
				if err := n.cmd.Command(n.vegaWallet, blockchain.NodeVoteCommand, nv); err != nil {
					n.log.Error("unable to send command", logging.Error(err))
					// @TODO keep in memory, retry later?
					continue
				}
			}
			// set new state so we do not try to validate again
			atomic.StoreUint32(&prop.validState, voteSentProposal)
		}
	}

	return accepted
}

// AddNodeVote registers a vote from a validator node for a given proposal
func (n *NodeValidation) AddNodeVote(nv *types.NodeVote) error {
	// get the node proposal first
	np, ok := n.nodeProposals[nv.Reference]
	if !ok {
		return ErrInvalidProposalReferenceForNodeVote
	}

	// ensure the node is a validator
	if !n.top.Exists(nv.PubKey) {
		n.log.Error("non-validator node tried to register node vote",
			logging.String("pubkey", hex.EncodeToString(nv.PubKey)))
		return ErrNodeIsNotAValidator
	}

	_, ok = np.votes[string(nv.PubKey)]
	if ok {
		return ErrDuplicateVoteFromNode
	}

	// add the vote
	np.votes[string(nv.PubKey)] = struct{}{}

	return nil
}

// IsNodeValidationRequired returns true if the given proposal require validation from a node.
func (n *NodeValidation) IsNodeValidationRequired(p *types.Proposal) bool {
	switch p.Terms.Change.(type) {
	case *types.ProposalTerms_NewAsset:
		return true
	default:
		return false
	}
}

// Start the node validation of a proposal
func (n *NodeValidation) Start(p *types.Proposal) error {
	if !n.IsNodeValidationRequired(p) {
		n.log.Error("no node validation required", logging.String("ref", p.Reference))
		return ErrNoNodeValidationRequired
	}

	_, ok := n.nodeProposals[p.Reference]
	if ok {
		return ErrProposalReferenceDuplicate
	}

	if err := n.checkProposal(p); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	np := &nodeProposal{
		Proposal:   p,
		votes:      map[string]struct{}{},
		validTime:  time.Unix(p.Terms.ValidationTimestamp, 0),
		validState: notValidatedProposal,
		cancel:     cancel,
	}
	n.nodeProposals[p.Reference] = np

	return n.start(ctx, np)
}

// start proposal specific validation and instanciation
func (n *NodeValidation) start(ctx context.Context, np *nodeProposal) error {
	// first initialize and underlying resources if needed
	// this can make error that we'll want to return for straaight away
	// next step will happen in goroutine.
	switch change := np.Terms.Change.(type) {
	case *types.ProposalTerms_NewAsset:
		assetID, err := n.assets.NewAsset(np.ID,
			change.NewAsset.GetChanges())
		if err != nil {
			n.log.Error("unable to instanciate asset",
				logging.String("asset-id", assetID),
				logging.Error(err))
			return err
		}

	default:
		// this should have been check earlier but in case of.
		return ErrNoNodeValidationRequired
	}

	// then start validations of the proposal specficis

	// if we are not a validator lets just assume it's valid
	if !n.isValidator {
		atomic.StoreUint32(&np.validState, validatedProposal)
	} else {
		// we are a validator so we need to make sure the proposal is valid
		switch np.Terms.Change.(type) {
		case *types.ProposalTerms_NewAsset:
			// start asset validation
			go n.validateAsset(ctx, np, np.Proposal)
		default:
			// this should have been check earlier but in case of.
			return ErrNoNodeValidationRequired
		}
	}
	return nil
}

func (n *NodeValidation) validateAsset(ctx context.Context, np *nodeProposal, prop *types.Proposal) {

	// get the asset to validate from the assets pool
	asset, err := n.assets.Get(prop.ID)
	// if we get an error here, we'll never change the state of the proposal,
	// so it will be dismissed later on by all the whole network
	if err != nil || asset == nil {
		n.log.Error("Validating asset, unable to get the asset",
			logging.String("ref", prop.GetTerms().String()),
			logging.Error(err),
		)
		return
	}

	// wait time between call to validation
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		// first try to validate the asset
		n.log.Debug("Validating asset",
			logging.String("asset-source", prop.GetTerms().String()),
		)

		// call validation
		err = asset.Validate()
		if err != nil {
			// we just log the error, but these are not criticals, as it may be
			// things unrelated to the current node, and would recover later on.
			// it's just informative
			n.log.Warn("error validating asset", logging.Error(err))
		} else {
			if asset.IsValid() {
				atomic.StoreUint32(&np.validState, validatedProposal)
				return
			}
		}

		// wait or break if the time's up
		select {
		case <-ctx.Done():
			n.log.Error("asset validation context done",
				logging.Error(ctx.Err()))
			return
		case _ = <-ticker.C:
		}
	}
}

func (n *NodeValidation) checkProposal(prop *types.Proposal) error {
	if prop.Terms.ClosingTimestamp < prop.Terms.ValidationTimestamp {
		return ErrProposalValidationTimestampInvalid
	}
	minValid, maxValid := n.currentTimestamp.Add(minValidationPeriod*time.Second), n.currentTimestamp.Add(maxValidationPeriod*time.Second)
	if prop.Terms.ValidationTimestamp < minValid.Unix() || prop.Terms.ValidationTimestamp > maxValid.Unix() {
		return ErrProposalValidationTimestampInvalid
	}
	return nil
}
