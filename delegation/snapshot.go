package delegation

import (
	"context"
	"errors"
	"strconv"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

var (
	hashKeys = []string{
		activeKey,
		pendingKey,
		autoKey,
	}

	ErrSnapshotKeyDoesNotExist  = errors.New("unknown key for delegation snapshot")
	ErrUnknownSnapshotType      = errors.New("snapshot data type not known")
	ErrInvalidSnapshotNamespace = errors.New("invalid snapshot namespace")
)

type delegationSnapshotState struct {
	changed    map[string]bool
	hash       map[string][]byte
	serialised map[string][]byte
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.DelegationSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) serialiseActive() ([]byte, error) {
	active := e.getActive()
	delegations := make([]*types.Delegation, 0, len(active))
	for _, a := range active {
		delegations = append(delegations, &types.Delegation{
			Party:    a.Party,
			NodeID:   a.Node,
			EpochSeq: strconv.FormatUint(a.EpochSeq, 10),
			Amount:   a.Amount.Clone(),
		})
	}

	payload := types.Payload{
		Data: &types.PayloadDelegationActive{
			DelegationActive: &types.DelegationActive{
				Delegations: delegations,
			},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialisePending() ([]byte, error) {
	pending := e.getPending()
	pendingDelegations := make([]*types.Delegation, 0, len(pending))
	pendingUndelegations := make([]*types.Delegation, 0, len(pending))
	for _, a := range pending {
		entry := &types.Delegation{
			Party:    a.Party,
			NodeID:   a.Node,
			EpochSeq: strconv.FormatUint(a.EpochSeq, 10),
			Amount:   a.Amount.Clone(),
		}
		if a.Undelegate {
			pendingUndelegations = append(pendingUndelegations, entry)
		} else {
			pendingDelegations = append(pendingDelegations, entry)
		}
	}
	payload := types.Payload{
		Data: &types.PayloadDelegationPending{
			DelegationPending: &types.DelegationPending{
				Delegations:  pendingDelegations,
				Undelegation: pendingUndelegations,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (e *Engine) serialiseAuto() ([]byte, error) {
	auto := e.getAuto()
	payload := types.Payload{
		Data: &types.PayloadDelegationAuto{
			DelegationAuto: &types.DelegationAuto{Parties: auto},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

// get the serialised form and hash of the given key
func (e *Engine) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if _, ok := e.keyToSerialiser[k]; !ok {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !e.dss.changed[k] {
		return e.dss.serialised[k], e.dss.hash[k], nil
	}

	data, err := e.keyToSerialiser[k]()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	e.dss.serialised[k] = data
	e.dss.hash[k] = hash
	e.dss.changed[k] = false
	return data, hash, nil
}

func (e *Engine) GetHash(k string) ([]byte, error) {
	_, hash, err := e.getSerialisedAndHash(k)
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, error) {
	state, _, err := e.getSerialisedAndHash(k)
	return state, err
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

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) error {
	if e.Namespace() != p.Data.Namespace() {
		return ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadDelegationActive:
		return e.restoreActive(ctx, pl.DelegationActive)
	case *types.PayloadDelegationPending:
		return e.restorePending(ctx, pl.DelegationPending)
	case *types.PayloadDelegationAuto:
		return e.restoreAuto(pl.DelegationAuto)
	default:
		return ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreActive(ctx context.Context, delegations *types.DelegationActive) error {
	entries := make([]*types.DelegationEntry, 0, len(delegations.Delegations))
	for _, d := range delegations.Delegations {
		epoch, _ := strconv.ParseUint(d.EpochSeq, 10, 64)
		entries = append(entries, &types.DelegationEntry{
			Party:    d.Party,
			Node:     d.NodeID,
			Amount:   d.Amount,
			EpochSeq: epoch,
		})
	}
	e.setActive(ctx, entries)
	// after reloading we need to set the dirty flag to true so that we know next time to recalc the hash/serialise
	e.dss.changed[activeKey] = true
	return nil
}

func (e *Engine) restorePending(ctx context.Context, delegations *types.DelegationPending) error {
	entries := make([]*types.DelegationEntry, 0, len(delegations.Delegations)+len(delegations.Undelegation))
	for _, d := range delegations.Delegations {
		epoch, _ := strconv.ParseUint(d.EpochSeq, 10, 64)
		entries = append(entries, &types.DelegationEntry{
			Party:    d.Party,
			Node:     d.NodeID,
			Amount:   d.Amount,
			EpochSeq: epoch,
		})
	}
	for _, d := range delegations.Undelegation {
		epoch, _ := strconv.ParseUint(d.EpochSeq, 10, 64)
		entries = append(entries, &types.DelegationEntry{
			Party:      d.Party,
			Node:       d.NodeID,
			Amount:     d.Amount,
			EpochSeq:   epoch,
			Undelegate: true,
		})
	}
	e.sortPending(entries)
	e.setPending(ctx, entries)
	// after reloading we need to set the dirty flag to true so that we know next time to recalc the hash/serialise
	e.dss.changed[pendingKey] = true
	return nil
}

func (e *Engine) restoreAuto(delegations *types.DelegationAuto) error {
	e.setAuto(delegations.Parties)
	// after reloading we need to set the dirty flag to true so that we know next time to recalc the hash/serialise
	e.dss.changed[autoKey] = true
	return nil
}
