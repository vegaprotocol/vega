package governance

import (
	"fmt"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/logging"
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
	minValidationPeriod = 1         // 1 sec
	maxValidationPeriod = 48 * 3600 // 2 days
	nodeApproval        = 1         // float for percentage
)

const (
	pendingValidationProposal uint32 = iota
	okProposal
	rejectedProposal
)

type NodeValidation struct {
	log              *logging.Logger
	assets           Assets
	currentTimestamp time.Time
	nodeProposals    map[string]*nodeProposal
	erc              ExtResChecker
}

type nodeProposal struct {
	*types.Proposal
	state   uint32
	checker func() error
}

func (n *nodeProposal) GetID() string {
	return fmt.Sprintf("proposal-node-validation-%v", n.ID)
}

func (n *nodeProposal) Check() error {
	return n.checker()
}

func NewNodeValidation(
	log *logging.Logger,
	assets Assets,
	now time.Time,
	erc ExtResChecker,
) (*NodeValidation, error) {
	return &NodeValidation{
		log:              log,
		nodeProposals:    map[string]*nodeProposal{},
		assets:           assets,
		currentTimestamp: now,
		erc:              erc,
	}, nil
}

func (n *NodeValidation) onResChecked(i interface{}, valid bool) {
	np, ok := i.(*nodeProposal)
	if !ok {
		n.log.Error("not an node proposal received from ext check")
		return
	}

	var newState = rejectedProposal
	if valid {
		newState = okProposal
	}
	atomic.StoreUint32(&np.state, newState)
}

// returns validated proposal by all nodes
func (n *NodeValidation) OnChainTimeUpdate(t time.Time) (accepted []*types.Proposal, rejected []*types.Proposal) {
	n.currentTimestamp = t

	// check that any proposal is ready
	for k, prop := range n.nodeProposals {
		// this proposal has passed the node-voting period, or all nodes have voted/approved
		// time expired, or all vote agregated, and own vote sent
		state := atomic.LoadUint32(&prop.state)
		if state == pendingValidationProposal {
			continue
		}

		switch state {
		case okProposal:
			accepted = append(accepted, prop.Proposal)
		case rejectedProposal:
			rejected = append(rejected, prop.Proposal)
		}
		delete(n.nodeProposals, k)
	}

	return accepted, rejected
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
		n.log.Error("no node validation required", logging.String("ref", p.ID))
		return ErrNoNodeValidationRequired
	}

	_, ok := n.nodeProposals[p.ID]
	if ok {
		return ErrProposalReferenceDuplicate
	}

	if err := n.checkProposal(p); err != nil {
		return err
	}

	checker, err := n.getChecker(p)
	if err != nil {
		return err
	}
	np := &nodeProposal{
		Proposal: p,
		state:    pendingValidationProposal,
		checker:  checker,
	}
	n.nodeProposals[p.ID] = np

	return n.erc.StartCheck(np, n.onResChecked, time.Unix(p.Terms.ValidationTimestamp, 0))
}

func (n *NodeValidation) getChecker(p *types.Proposal) (func() error, error) {
	switch change := p.Terms.Change.(type) {
	case *types.ProposalTerms_NewAsset:
		assetID, err := n.assets.NewAsset(p.ID,
			change.NewAsset.GetChanges())
		if err != nil {
			n.log.Error("unable to instanciate asset",
				logging.String("asset-id", assetID),
				logging.Error(err))
			return nil, err
		}
		return func() error {
			return n.checkAsset(p.ID)
		}, nil
	default: // this should have been check earlier but in case of.
		return nil, ErrNoNodeValidationRequired
	}
}

func (n *NodeValidation) checkAsset(assetID string) error {
	// get the asset to validate from the assets pool
	asset, err := n.assets.Get(assetID)
	// if we get an error here, we'll never change the state of the proposal,
	// so it will be dismissed later on by all the whole network
	if err != nil || asset == nil {
		n.log.Error("Validating asset, unable to get the asset",
			logging.String("id", assetID),
			logging.Error(err),
		)
		return errors.New("invalid asset ID")
	}

	err = asset.Validate()
	if err != nil {
		// we just log the error, but these are not criticals, as it may be
		// things unrelated to the current node, and would recover later on.
		// it's just informative
		n.log.Warn("error validating asset", logging.Error(err))
		return err
	}
	if asset.IsValid() {
		return nil
	}
	return nil
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
