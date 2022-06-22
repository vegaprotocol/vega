// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	activeKey         = (&types.PayloadGovernanceActive{}).Key()
	enactedKey        = (&types.PayloadGovernanceEnacted{}).Key()
	nodeValidationKey = (&types.PayloadGovernanceNode{}).Key()

	hashKeys = []string{
		activeKey,
		enactedKey,
		nodeValidationKey,
	}
)

type governanceSnapshotState struct {
	serialisedActive         []byte
	serialisedEnacted        []byte
	serialisedNodeValidation []byte
	changedActive            bool
	changedEnacted           bool
	changedNodeValidation    bool
}

// serialiseActiveProposals returns the engine's active proposals as marshalled bytes.
func (e *Engine) serialiseActiveProposals() ([]byte, error) {
	pending := make([]*types.ProposalData, 0, len(e.activeProposals))
	for _, p := range e.activeProposals {
		pp := &types.ProposalData{
			Proposal: p.Proposal,
			Yes:      votesAsSlice(p.yes),
			No:       votesAsSlice(p.no),
			Invalid:  votesAsSlice(p.invalidVotes),
		}
		pending = append(pending, pp)
	}

	pl := types.Payload{
		Data: &types.PayloadGovernanceActive{
			GovernanceActive: &types.GovernanceActive{
				Proposals: pending,
			},
		},
	}

	return proto.Marshal(pl.IntoProto())
}

// serialiseEnactedProposals returns the engine's enacted proposals as marshalled bytes.
func (e *Engine) serialiseEnactedProposals() ([]byte, error) {
	enacted := make([]*types.ProposalData, 0, len(e.activeProposals))
	for _, p := range e.enactedProposals {
		pp := &types.ProposalData{
			Proposal: p.Proposal,
			Yes:      votesAsSlice(p.yes),
			No:       votesAsSlice(p.no),
			Invalid:  votesAsSlice(p.invalidVotes),
		}
		enacted = append(enacted, pp)
	}

	pl := types.Payload{
		Data: &types.PayloadGovernanceEnacted{
			GovernanceEnacted: &types.GovernanceEnacted{
				Proposals: enacted,
			},
		},
	}
	return proto.Marshal(pl.IntoProto())
}

// serialiseNodeProposals returns the engine's proposals waiting for node validation.
func (e *Engine) serialiseNodeProposals() ([]byte, error) {
	nodeProposals := e.nodeProposalValidation.getProposals()
	proposals := make([]*types.Proposal, 0, len(nodeProposals))

	for _, np := range nodeProposals {
		// Given a snapshot is always taken at the end of a block the value of `state` in np will
		// always be pending since any that are not will have already been resolved as accepted/rejected
		// and removed from the slice. The yes/no/invalid fields in `np.proposal` are also unnecessary to
		// save since "voting" as is done for active proposals is not done on node-proposals, and so the
		// maps will always be empty
		p := np.proposal.Proposal
		proposals = append(proposals, p)
	}

	pl := types.Payload{
		Data: &types.PayloadGovernanceNode{
			GovernanceNode: &types.GovernanceNode{
				Proposals: proposals,
			},
		},
	}
	return proto.Marshal(pl.IntoProto())
}

func (e *Engine) serialiseK(k string, serialFunc func() ([]byte, error), dataField *[]byte, changedField *bool) ([]byte, error) {
	if !e.HasChanged(k) {
		if dataField == nil {
			return nil, nil
		}
		return *dataField, nil
	}
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	*changedField = false
	return data, nil
}

func (e *Engine) serialise(k string) ([]byte, error) {
	switch k {
	case activeKey:
		return e.serialiseK(k, e.serialiseActiveProposals, &e.gss.serialisedActive, &e.gss.changedActive)
	case enactedKey:
		return e.serialiseK(k, e.serialiseEnactedProposals, &e.gss.serialisedEnacted, &e.gss.changedEnacted)
	case nodeValidationKey:
		return e.serialiseK(k, e.serialiseNodeProposals, &e.gss.serialisedNodeValidation, &e.gss.changedNodeValidation)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.GovernanceSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

func (e *Engine) HasChanged(k string) bool {
	switch k {
	case activeKey:
		return e.gss.changedActive
	case enactedKey:
		return e.gss.changedEnacted
	case nodeValidationKey:
		return e.gss.changedNodeValidation
	default:
		return false
	}
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := e.serialise(k)
	return data, nil, err
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := p.Data.(type) {
	case *types.PayloadGovernanceActive:
		return nil, e.restoreActiveProposals(ctx, pl.GovernanceActive, p)
	case *types.PayloadGovernanceEnacted:
		return nil, e.restoreEnactedProposals(ctx, pl.GovernanceEnacted, p)
	case *types.PayloadGovernanceNode:
		return nil, e.restoreNodeProposals(ctx, pl.GovernanceNode, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreActiveProposals(ctx context.Context, active *types.GovernanceActive, p *types.Payload) error {
	e.activeProposals = make([]*proposal, 0, len(active.Proposals))
	evts := []events.Event{}
	vevts := []events.Event{}
	e.log.Debug("restoring active proposals snapshot", logging.Int("nproposals", len(active.Proposals)))
	for _, p := range active.Proposals {
		pp := &proposal{
			Proposal:     p.Proposal,
			yes:          votesAsMap(p.Yes),
			no:           votesAsMap(p.No),
			invalidVotes: votesAsMap(p.Invalid),
		}
		e.log.Debug("proposals",
			logging.String("id", pp.ID),
			logging.Int("yes", len(pp.yes)),
			logging.Int("no", len(pp.no)),
			logging.Int("invalid", len(pp.invalidVotes)),
		)
		e.activeProposals = append(e.activeProposals, pp)
		evts = append(evts, events.NewProposalEvent(ctx, *pp.Proposal))

		for _, v := range pp.yes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
		for _, v := range pp.no {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}

		for _, v := range pp.invalidVotes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
	}

	var err error
	e.gss.changedActive = false
	e.gss.serialisedActive, err = proto.Marshal(p.IntoProto())
	e.broker.SendBatch(evts)
	e.broker.SendBatch(vevts)
	return err
}

func (e *Engine) restoreEnactedProposals(ctx context.Context, enacted *types.GovernanceEnacted, p *types.Payload) error {
	evts := []events.Event{}
	vevts := []events.Event{}
	e.log.Debug("restoring enacted proposals snapshot", logging.Int("nproposals", len(enacted.Proposals)))
	for _, p := range enacted.Proposals {
		pp := &proposal{
			Proposal:     p.Proposal,
			yes:          votesAsMap(p.Yes),
			no:           votesAsMap(p.No),
			invalidVotes: votesAsMap(p.Invalid),
		}
		e.log.Debug("proposals",
			logging.String("id", pp.ID),
			logging.Int("yes", len(pp.yes)),
			logging.Int("no", len(pp.no)),
			logging.Int("invalid", len(pp.invalidVotes)),
		)
		e.enactedProposals = append(e.enactedProposals, pp)
		evts = append(evts, events.NewProposalEvent(ctx, *pp.Proposal))

		for _, v := range pp.yes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
		for _, v := range pp.no {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}

		for _, v := range pp.invalidVotes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
	}
	var err error
	e.gss.changedEnacted = false
	e.gss.serialisedEnacted, _ = proto.Marshal(p.IntoProto())
	e.broker.SendBatch(evts)
	e.broker.SendBatch(vevts)

	return err
}

func (e *Engine) restoreNodeProposals(ctx context.Context, node *types.GovernanceNode, p *types.Payload) error {
	for _, p := range node.Proposals {
		e.nodeProposalValidation.restore(p)
		e.broker.Send(events.NewProposalEvent(ctx, *p))
	}
	var err error
	e.gss.changedNodeValidation = false
	e.gss.serialisedNodeValidation, err = proto.Marshal(p.IntoProto())
	return err
}

// votesAsSlice returns a sorted slice of votes from a given map of votes.
func votesAsSlice(votes map[string]*types.Vote) []*types.Vote {
	ret := make([]*types.Vote, 0, len(votes))
	for _, v := range votes {
		ret = append(ret, v)
	}
	sort.SliceStable(ret, func(i, j int) bool { return ret[i].PartyID < ret[j].PartyID })
	return ret
}

// votesAsMap returns an partyID => Vote map from the given slice of votes.
func votesAsMap(votes []*types.Vote) map[string]*types.Vote {
	r := make(map[string]*types.Vote, len(votes))
	for _, v := range votes {
		r[v.PartyID] = v
	}
	return r
}
