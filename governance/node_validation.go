package governance

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	ErrNoNodeValidationRequired   = errors.New("no node validation required")
	ErrProposalReferenceDuplicate = errors.New("proposal duplicate")
)

const (
	minValidationPeriod = 600       // ten minutes
	maxValidationPeriod = 48 * 3600 // 2 days
	nodeApproval        = 1         // float for percentage
)

type NodeValidation struct {
	// used for nodes proposals
	top           ValidatorTopology
	nodeProposals map[string]*nodeProposal

	currentTimestamp time.Time
}

type nodeProposal struct {
	*types.Proposal
	votes     map[string]struct{}
	validTime time.Time
	// use for the node internal validation
	validState uint32
	cancel     func()
}

func (n *NodeValidation) newNodeValidation(
	top ValidatorTopology,
	now time.Time,
) *NodeValidation {
	return &NodeValidation{
		top:              top,
		nodeProposals:    map[string]*nodeProposal{},
		currentTimestamp: now,
	}
}

// returns validated proposal by all nodes
func (n *NodeValidation) OnChainTimeUpdate(t time.Time) {
	n.currentTimestamp = t
}

func (n *NodeValidation) IsNodeValidationRequired(p *types.Proposal) bool {
	if na := proposal.Terms.GetNewAsset(); na != nil {
		return true
	}
	// add more cases here if needed later.
	return false
}

func (n *NodeValidation) Start(p *types.Proposal) error {
	if !n.IsNodeValidationRequired(p) {
		p.log.Error("not an asset proposal", logging.String("ref", proposal.Reference))
		return ErrNoNodeValidationRequired
	}

	_, ok := p.nodeProposals[proposal.Reference]
	if ok {
		return ErrProposalReferenceDuplicate
	}

	if err := p.checkProposal(proposal); err != nil {
		return err
	}

	assetID, err := p.assets.NewAsset(proposal.Reference,
		proposal.Terms.GetNewAsset().GetChanges())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	// @TODO check valid timestamps
	np := &nodeProposal{
		Proposal:   proposal,
		votes:      map[string]struct{}{},
		validTime:  time.Unix(proposal.Terms.ValidationTimestamp, 0),
		validState: notValidAssetProposal,
		cancel:     cancel,
		assetID:    assetID,
	}
	p.nodeProposals[proposal.Reference] = np
	// start asset validation
	go p.validateAsset(ctx, np, proposal)

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
