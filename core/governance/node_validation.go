// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package governance

import (
	"context"
	"encoding/binary"
	"errors"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrNoNodeValidationRequired           = errors.New("no node validation required")
	ErrProposalReferenceDuplicate         = errors.New("proposal duplicate")
	ErrProposalValidationTimestampInvalid = errors.New("proposal validation timestamp invalid")
)

const (
	minValidationPeriod = 1         // 1 sec
	maxValidationPeriod = 48 * 3600 // 2 days

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
	nodeProposals    []*nodeProposal
	witness          Witness
}

type nodeProposal struct {
	*proposal
	state   atomic.Uint32
	checker func() error
}

func (n *nodeProposal) GetID() string {
	return n.ID
}

func (n *nodeProposal) GetType() types.NodeVoteType {
	return types.NodeVoteTypeGovernanceValidateAsset
}

func (n *nodeProposal) Check() error {
	// always keep the last error
	err := n.checker()
	if err != nil {
		return err
	}

	return nil
}

func NewNodeValidation(
	log *logging.Logger,
	assets Assets,
	now time.Time,
	witness Witness,
) *NodeValidation {
	return &NodeValidation{
		log:              log,
		nodeProposals:    []*nodeProposal{},
		assets:           assets,
		currentTimestamp: now,
		witness:          witness,
	}
}

func (n *NodeValidation) Hash() []byte {
	// 32 -> len(proposal.ID) = 32 bytes pubkey
	// vote counts = 3*uint64
	output := make([]byte, len(n.nodeProposals)*(32+8*3))
	var i int
	for _, k := range n.nodeProposals {
		idbytes := []byte(k.ID)
		copy(output[i:], idbytes[:])
		i += 32
		binary.BigEndian.PutUint64(output[i:], uint64(len(k.yes)))
		i += 8
		binary.BigEndian.PutUint64(output[i:], uint64(len(k.no)))
		i += 8
		binary.BigEndian.PutUint64(output[i:], uint64(len(k.invalidVotes)))
		i += 8
	}

	return vgcrypto.Hash(output)
}

func (n *NodeValidation) onResChecked(i interface{}, valid bool) {
	np, ok := i.(*nodeProposal)
	if !ok {
		n.log.Error("not an node proposal received from ext check")
		return
	}

	newState := rejectedProposal
	if valid {
		newState = okProposal
	}
	np.state.Store(newState)
}

func (n *NodeValidation) getProposal(id string) (*nodeProposal, bool) {
	for _, v := range n.nodeProposals {
		if v.ID == id {
			return v, true
		}
	}
	return nil, false
}

func (n *NodeValidation) getProposals() []*nodeProposal {
	return n.nodeProposals
}

func (n *NodeValidation) removeProposal(id string) {
	for i, p := range n.nodeProposals {
		if p.ID == id {
			copy(n.nodeProposals[i:], n.nodeProposals[i+1:])
			n.nodeProposals[len(n.nodeProposals)-1] = nil
			n.nodeProposals = n.nodeProposals[:len(n.nodeProposals)-1]
			return
		}
	}
}

// OnTick returns validated proposal by all nodes.
func (n *NodeValidation) OnTick(t time.Time) (accepted []*proposal, rejected []*proposal) { //revive:disable:unexported-return
	n.currentTimestamp = t

	toRemove := []string{} // id of proposals to remove

	// check that any proposal is ready
	for _, prop := range n.nodeProposals {
		// this proposal has passed the node-voting period, or all nodes have voted/approved
		// time expired, or all vote aggregated, and own vote sent
		switch prop.state.Load() {
		case pendingValidationProposal:
			continue
		case okProposal:
			accepted = append(accepted, prop.proposal)
		case rejectedProposal:
			rejected = append(rejected, prop.proposal)
		}
		toRemove = append(toRemove, prop.ID)
	}

	// now we iterate over all proposal ids to remove them from the list
	for _, id := range toRemove {
		n.removeProposal(id)
	}

	return accepted, rejected
}

// IsNodeValidationRequired returns true if the given proposal require validation from a node.
func (n *NodeValidation) IsNodeValidationRequired(p *types.Proposal) bool {
	switch p.Terms.Change.(type) {
	case *types.ProposalTermsNewAsset:
		return true
	default:
		return false
	}
}

// Start the node validation of a proposal.
func (n *NodeValidation) Start(ctx context.Context, p *types.Proposal) error {
	if !n.IsNodeValidationRequired(p) {
		n.log.Error("no node validation required", logging.String("ref", p.ID))
		return ErrNoNodeValidationRequired
	}

	if _, ok := n.getProposal(p.ID); ok {
		return ErrProposalReferenceDuplicate
	}

	if err := n.checkProposal(p); err != nil {
		return err
	}

	checker, err := n.getChecker(ctx, p)
	if err != nil {
		return err
	}

	np := &nodeProposal{
		proposal: &proposal{
			Proposal:     p,
			yes:          map[string]*types.Vote{},
			no:           map[string]*types.Vote{},
			invalidVotes: map[string]*types.Vote{},
		},
		state:   atomic.Uint32{},
		checker: checker,
	}
	np.state.Store(pendingValidationProposal)
	n.nodeProposals = append(n.nodeProposals, np)

	return n.witness.StartCheck(
		np, n.onResChecked, time.Unix(p.Terms.ValidationTimestamp, 0))
}

func (n *NodeValidation) restore(ctx context.Context, p *types.Proposal) error {
	checker, err := n.getChecker(ctx, p)
	if err != nil {
		return err
	}
	np := &nodeProposal{
		proposal: &proposal{
			Proposal:     p,
			yes:          map[string]*types.Vote{},
			no:           map[string]*types.Vote{},
			invalidVotes: map[string]*types.Vote{},
		},
		state:   atomic.Uint32{},
		checker: checker,
	}
	np.state.Store(pendingValidationProposal)
	n.nodeProposals = append(n.nodeProposals, np)
	if err := n.witness.RestoreResource(np, n.onResChecked); err != nil {
		n.log.Panic("unable to restore witness resource", logging.String("id", np.ID), logging.Error(err))
	}
	return nil
}

func (n *NodeValidation) getChecker(ctx context.Context, p *types.Proposal) (func() error, error) {
	switch change := p.Terms.Change.(type) {
	case *types.ProposalTermsNewAsset:
		assetID, err := n.assets.NewAsset(ctx, p.ID, change.NewAsset.GetChanges())
		if err != nil {
			n.log.Error("unable to instantiate asset",
				logging.AssetID(assetID),
				logging.Error(err))
			return nil, err
		}
		return func() error {
			return n.checkAsset(p.ID)
		}, nil
	default: // this should have been checked earlier but in case of.
		return nil, ErrNoNodeValidationRequired
	}
}

func (n *NodeValidation) checkAsset(assetID string) error {
	err := n.assets.ValidateAsset(assetID)
	if err != nil {
		// we just log the error, but these are not critical, as it may be
		// things unrelated to the current node, and would recover later on.
		// it's just informative
		n.log.Warn("error validating asset", logging.Error(err))
	}
	return err
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
