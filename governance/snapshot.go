package governance

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
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
	hash       map[string][]byte
	serialised map[string][]byte
	changed    map[string]bool
}

// serialiseActiveProposals returns the engine's active proposals as marshalled bytes.
func (e *Engine) serialiseActiveProposals() ([]byte, error) {
	pending := make([]*types.PendingProposal, 0, len(e.activeProposals))
	for _, p := range e.activeProposals {
		pp := &types.PendingProposal{
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
	pl := types.Payload{
		Data: &types.PayloadGovernanceEnacted{
			GovernanceEnacted: &types.GovernanceEnacted{
				Proposals: e.enactedProposals,
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

// get the serialised form and hash of the given key.
func (e *Engine) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if _, ok := e.keyToSerialiser[k]; !ok {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !e.gss.changed[k] {
		return e.gss.serialised[k], e.gss.hash[k], nil
	}

	data, err := e.keyToSerialiser[k]()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	e.gss.serialised[k] = data
	e.gss.hash[k] = hash
	e.gss.changed[k] = false
	return data, hash, nil
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.GovernanceSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) GetHash(k string) ([]byte, error) {
	_, hash, err := e.getSerialisedAndHash(k)
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, error) {
	data, _, err := e.getSerialisedAndHash(k)
	return data, err
}

func (e *Engine) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadGovernanceActive:
		return nil, e.restoreActiveProposals(pl.GovernanceActive)
	case *types.PayloadGovernanceEnacted:
		return nil, e.restoreEnactedProposals(pl.GovernanceEnacted)
	case *types.PayloadGovernanceNode:
		return nil, e.restoreNodeProposals(pl.GovernanceNode)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreActiveProposals(active *types.GovernanceActive) error {
	e.activeProposals = make([]*proposal, 0, len(active.Proposals))
	for _, p := range active.Proposals {
		pp := &proposal{
			Proposal:     p.Proposal,
			yes:          votesAsMap(p.Yes),
			no:           votesAsMap(p.No),
			invalidVotes: votesAsMap(p.Invalid),
		}

		e.activeProposals = append(e.activeProposals, pp)
	}

	e.gss.changed[activeKey] = true
	return nil
}

func (e *Engine) restoreEnactedProposals(enacted *types.GovernanceEnacted) error {
	e.enactedProposals = enacted.Proposals[:]
	e.gss.changed[enactedKey] = true
	return nil
}

func (e *Engine) restoreNodeProposals(node *types.GovernanceNode) error {
	for _, p := range node.Proposals {
		e.nodeProposalValidation.restore(p)
	}
	e.gss.changed[nodeValidationKey] = true
	return nil
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
