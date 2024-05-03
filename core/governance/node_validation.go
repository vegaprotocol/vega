// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package governance

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgerrors "code.vegaprotocol.io/vega/libs/errors"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

const (
	minValidationPeriod = 1         // 1 sec
	maxValidationPeriod = 48 * 3600 // 2 days
)

var (
	ErrNoNodeValidationRequired                = errors.New("no node validation required")
	ErrProposalReferenceDuplicate              = errors.New("proposal duplicate")
	ErrProposalValidationTimestampTooLate      = errors.New("proposal validation timestamp must be earlier than closing time")
	ErrProposalValidationTimestampOutsideRange = fmt.Errorf("proposal validation timestamp must be within %d-%d seconds from submission time", minValidationPeriod, maxValidationPeriod)
)

const (
	pendingValidationProposal uint32 = iota
	okProposal
	rejectedProposal
)

type NodeValidation struct {
	log                *logging.Logger
	assets             Assets
	currentTimestamp   time.Time
	nodeProposals      []*nodeProposal
	nodeBatchProposals []*nodeBatchProposal
	witness            Witness
}

type nodeProposal struct {
	*proposal
	state   atomic.Uint32
	checker func() error
}

type nodeBatchProposal struct {
	*batchProposal
	nodeProposals []*nodeProposal
	state         atomic.Uint32
}

func (n *nodeBatchProposal) UpdateState() {
	pending, failed := 0, 0

	for _, v := range n.nodeProposals {
		switch v.state.Load() {
		case okProposal:
			continue
		case pendingValidationProposal:
			pending++
		case rejectedProposal:
			failed++
		}
	}

	// nothing to do
	if pending > 0 {
		return
	}

	// if at least 1 failure and no pending, then the whole batch is failed.
	if failed > 0 {
		n.state.Store(rejectedProposal)
		return
	}

	n.state.Store(okProposal)
}

func (n *nodeProposal) GetID() string {
	return n.ID
}

func (n *nodeProposal) GetChainID() string {
	switch na := n.Terms.Change.(type) {
	case *types.ProposalTermsNewAsset:
		if erc20 := na.NewAsset.Changes.GetERC20(); erc20 != nil {
			return erc20.ChainID
		}
	}
	return ""
}

func (n *nodeProposal) GetType() types.NodeVoteType {
	return types.NodeVoteTypeGovernanceValidateAsset
}

func (n *nodeProposal) Check(_ context.Context) error {
	if err := n.checker(); err != nil {
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
		log:                log,
		nodeProposals:      []*nodeProposal{},
		nodeBatchProposals: []*nodeBatchProposal{},
		assets:             assets,
		currentTimestamp:   now,
		witness:            witness,
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

func (n *NodeValidation) getBatchProposal(id string) (*nodeBatchProposal, bool) {
	for _, v := range n.nodeBatchProposals {
		if v.ID == id {
			return v, true
		}
	}
	return nil, false
}

func (n *NodeValidation) getProposals() []*nodeProposal {
	return n.nodeProposals
}

func (n *NodeValidation) getBatchProposals() []*nodeBatchProposal {
	return n.nodeBatchProposals
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

func (n *NodeValidation) removeBatchProposal(id string) {
	for i, p := range n.nodeBatchProposals {
		if p.ID == id {
			copy(n.nodeBatchProposals[i:], n.nodeBatchProposals[i+1:])
			n.nodeBatchProposals[len(n.nodeBatchProposals)-1] = nil
			n.nodeBatchProposals = n.nodeBatchProposals[:len(n.nodeBatchProposals)-1]
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

// OnTickBatch returns validated proposal by all nodes.
func (n *NodeValidation) OnTickBatch(t time.Time) (accepted []*batchProposal, rejected []*batchProposal) { //revive:disable:unexported-return
	n.currentTimestamp = t

	toRemove := []string{} // id of proposals to remove

	// check that any proposal is ready
	for _, prop := range n.nodeBatchProposals {
		// update the top level batch proposal
		prop.UpdateState()
		// this proposal has passed the node-voting period, or all nodes have voted/approved
		// time expired, or all vote aggregated, and own vote sent
		switch prop.state.Load() {
		case pendingValidationProposal:
			continue
		case okProposal:
			accepted = append(accepted, prop.batchProposal)
		case rejectedProposal:
			rejected = append(rejected, prop.batchProposal)
		}
		toRemove = append(toRemove, prop.ID)
	}

	// now we iterate over all proposal ids to remove them from the list
	for _, id := range toRemove {
		n.removeBatchProposal(id)
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

func (n *NodeValidation) IsNodeValidationRequiredBatch(p *types.BatchProposal) (is bool) {
	for _, v := range p.Proposals {
		is = is || n.IsNodeValidationRequired(v)
	}

	return is
}

// Start the node validation of a proposal.
func (n *NodeValidation) StartBatch(ctx context.Context, p *types.BatchProposal) error {
	if !n.IsNodeValidationRequiredBatch(p) {
		n.log.Error("no node validation required", logging.String("ref", p.ID))
		return ErrNoNodeValidationRequired
	}

	if _, ok := n.getBatchProposal(p.ID); ok {
		return ErrProposalReferenceDuplicate
	}

	if err := n.checkBatchProposal(p); err != nil {
		return err
	}

	nodeProposals := []*nodeProposal{}
	for _, v := range p.Proposals {
		if !n.IsNodeValidationRequired(v) {
			// nothing to do here
			continue
		}
		checker, err := n.getChecker(ctx, v)
		if err != nil {
			return err
		}

		np := &nodeProposal{
			proposal: &proposal{
				Proposal:     v,
				yes:          map[string]*types.Vote{},
				no:           map[string]*types.Vote{},
				invalidVotes: map[string]*types.Vote{},
			},
			state:   atomic.Uint32{},
			checker: checker,
		}

		np.state.Store(pendingValidationProposal)
		nodeProposals = append(nodeProposals, np)
	}

	nbp := &nodeBatchProposal{
		batchProposal: &batchProposal{
			BatchProposal: p,
			yes:           map[string]*types.Vote{},
			no:            map[string]*types.Vote{},
			invalidVotes:  map[string]*types.Vote{},
		},
		nodeProposals: nodeProposals,
		state:         atomic.Uint32{},
	}
	nbp.state.Store(pendingValidationProposal)
	n.nodeBatchProposals = append(n.nodeBatchProposals, nbp)

	errs := vgerrors.NewCumulatedErrors()
	for _, v := range nbp.nodeProposals {
		err := n.witness.StartCheck(v, n.onResChecked, time.Unix(v.Terms.ValidationTimestamp, 0))
		if err != nil {
			errs.Add(err)
		}
	}

	if errs.HasAny() {
		return errs
	}

	return nil
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

	return n.witness.StartCheck(np, n.onResChecked, time.Unix(p.Terms.ValidationTimestamp, 0))
}

func (n *NodeValidation) restoreBatch(ctx context.Context, pProto *snapshotpb.BatchProposalData) (*types.BatchProposal, error) {
	p := types.BatchProposalFromSnapshotProto(pProto.BatchProposal.Proposal, pProto.Proposals)
	nodeProposals := []*nodeProposal{}
	for _, v := range p.Proposals {
		if !n.IsNodeValidationRequired(v) {
			// nothing to do here
			continue
		}
		checker, err := n.getChecker(ctx, v)
		if err != nil {
			return nil, err
		}

		np := &nodeProposal{
			proposal: &proposal{
				Proposal:     v,
				yes:          map[string]*types.Vote{},
				no:           map[string]*types.Vote{},
				invalidVotes: map[string]*types.Vote{},
			},
			state:   atomic.Uint32{},
			checker: checker,
		}

		np.state.Store(pendingValidationProposal)
		nodeProposals = append(nodeProposals, np)
	}

	nbp := &nodeBatchProposal{
		batchProposal: &batchProposal{
			BatchProposal: p,
			yes:           map[string]*types.Vote{},
			no:            map[string]*types.Vote{},
			invalidVotes:  map[string]*types.Vote{},
		},
		nodeProposals: nodeProposals,
		state:         atomic.Uint32{},
	}

	nbp.state.Store(pendingValidationProposal)
	n.nodeBatchProposals = append(n.nodeBatchProposals, nbp)

	for _, v := range nbp.nodeProposals {
		if err := n.witness.RestoreResource(v, n.onResChecked); err != nil {
			n.log.Panic("unable to restore witness resource", logging.String("id", v.ID), logging.Error(err))
		}
	}

	return p, nil
}

func (n *NodeValidation) restore(ctx context.Context, p *types.ProposalData) error {
	checker, err := n.getChecker(ctx, p.Proposal)
	if err != nil {
		return err
	}
	np := &nodeProposal{
		proposal: &proposal{
			Proposal:     p.Proposal,
			yes:          votesAsMap(p.Yes),
			no:           votesAsMap(p.No),
			invalidVotes: votesAsMap(p.Invalid),
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
		return ErrProposalValidationTimestampTooLate
	}
	minValid, maxValid := n.currentTimestamp.Add(minValidationPeriod*time.Second), n.currentTimestamp.Add(maxValidationPeriod*time.Second)
	if prop.Terms.ValidationTimestamp < minValid.Unix() || prop.Terms.ValidationTimestamp > maxValid.Unix() {
		return ErrProposalValidationTimestampOutsideRange
	}
	return nil
}

func (n *NodeValidation) checkBatchProposal(prop *types.BatchProposal) error {
	for _, v := range prop.Proposals {
		if !n.IsNodeValidationRequired(v) {
			continue
		}

		if err := n.checkProposal(v); err != nil {
			return err
		}
	}

	return nil
}
