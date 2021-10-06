package governance

import (
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

var (
	activeKey  = (&types.PayloadGovernanceActive{}).Key()
	enactedKey = (&types.PayloadGovernanceEnacted{}).Key()
	nodeKey    = (&types.PayloadGovernanceNode{}).Key()

	hashKeys = []string{
		activeKey,
		enactedKey,
		nodeKey,
	}
)

type governanceSnapshotState struct {
	hash       map[string][]byte
	serialised map[string][]byte
	changed    map[string]bool
}

func sortedVotes(votes map[string]*types.Vote) []*types.Vote {
	ret := make([]*types.Vote, 0, len(votes))
	for _, v := range votes {
		ret = append(ret, v)
	}

	sort.SliceStable(ret, func(i, j int) bool { return ret[i].PartyID < ret[j].PartyID })
	return ret
}

func (e *Engine) serialiseActive() ([]byte, error) {

	if !e.gss.changed[activeKey] {
		return nil, nil
	}

	pending := make([]*types.PendingProposal, 0, len(e.activeProposals))
	for _, p := range e.activeProposals {

		pp := &types.PendingProposal{
			Proposal: p.Proposal,
			Yes:      sortedVotes(p.yes),
			No:       sortedVotes(p.no),
			Invalid:  sortedVotes(p.invalidVotes),
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

func (e *Engine) serialiseEnacted() ([]byte, error) {

	if !e.gss.changed[enactedKey] {
		return nil, nil
	}

	pl := types.Payload{
		Data: &types.PayloadGovernanceEnacted{
			GovernanceEnacted: &types.GovernanceEnacted{
				Proposals: e.enactedProposals,
			},
		},
	}
	return proto.Marshal(pl.IntoProto())
}

func (e *Engine) serialiseNode() ([]byte, error) {

	if !e.gss.changed[nodeKey] {
		return nil, nil
	}

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

// get the serialised form and hash of the given key
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
	return []string{activeKey, enactedKey}
}

func (e *Engine) GetHash(k string) ([]byte, error) {
	_, hash, err := e.getSerialisedAndHash(k)
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, error) {
	data, _, err := e.getSerialisedAndHash(k)
	return data, err
}

func (e *Engine) Snapshot() (map[string][]byte, error) {
	r := make(map[string][]byte, len(hashKeys))
	for _, k := range hashKeys {
		state, err := e.GetState(k)
		if err != nil {
			return nil, err
		}
		r[k] = state
	}
	return r, nil
}

func (e *Engine) LoadState(payload *types.Payload) error {

	if e.Namespace() != payload.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadGovernanceActive:
		return e.restoreActive(pl.GovernanceActive)
	case *types.PayloadGovernanceEnacted:
		return e.restoreEnacted(pl.GovernanceEnacted)
	case *types.PayloadGovernanceNode:
		return e.restoreNode(pl.GovernanceNode)
	default:
		return types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreActive(active *types.GovernanceActive) error {

	e.activeProposals = make([]*proposal, 0, len(active.Proposals))
	for _, p := range active.Proposals {

		pp := &proposal{
			Proposal:     p.Proposal,
			yes:          unpackVotes(p.Yes),
			no:           unpackVotes(p.No),
			invalidVotes: unpackVotes(p.Invalid),
		}

		e.activeProposals = append(e.activeProposals, pp)
	}

	e.gss.changed[activeKey] = true
	return nil
}

func (e *Engine) restoreEnacted(enacted *types.GovernanceEnacted) error {
	e.enactedProposals = enacted.Proposals[:]
	e.gss.changed[enactedKey] = true
	return nil
}

func (e *Engine) restoreNode(node *types.GovernanceNode) error {

	for _, p := range node.Proposals {
		e.nodeProposalValidation.Start(p)
	}
	e.gss.changed[nodeKey] = true
	return nil
}

func unpackVotes(votes []*types.Vote) map[string]*types.Vote {
	r := make(map[string]*types.Vote, len(votes))
	for _, v := range votes {
		r[v.PartyID] = v
	}
	return r
}
