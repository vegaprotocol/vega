package delegation

import (
	"errors"
	"strconv"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

var (
	hashKeys = []string{
		"active",
		"pending",
		"auto",
	}

	ErrSnapshotKeyDoesNotExist  = errors.New("unknown key for delegation snapshot")
	ErrUnknownSnapshotType      = errors.New("snapshot data type not known")
	ErrInvalidSnapshotNamespace = errors.New("invalid snapshot namespace")
)

type delegationSnapshotState struct {
	pendingChanged bool
	activeChanged  bool
	autoChanged    bool

	pendingHash []byte
	activeHash  []byte
	autoHash    []byte

	serialisedPending []byte
	serialisedActive  []byte
	serialisedAuto    []byte
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.DelegationSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

// recalculate active hash/serialise if the state has changed and return the active serialised form and hash
func (e *Engine) updateActiveAndGet() ([]byte, []byte, error) {
	if !e.dss.activeChanged {
		return e.dss.serialisedActive, e.dss.activeHash, nil
	}
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
	data, err := proto.Marshal(types.DelegationActive{Delegations: delegations}.IntoProto())
	if err != nil {
		return nil, nil, err
	}

	e.dss.serialisedActive = data
	e.dss.activeHash = crypto.Hash(data)
	e.dss.activeChanged = false
	return e.dss.serialisedActive, e.dss.activeHash, nil
}

// recalculate pending hash/serialise if the state has changed and return the pending serialised form and hash
func (e *Engine) updatePendingAndGet() ([]byte, []byte, error) {
	if !e.dss.pendingChanged {
		return e.dss.serialisedPending, e.dss.pendingHash, nil
	}
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
	data, err := proto.Marshal(types.DelegationPending{Delegations: pendingDelegations, Undelegation: pendingUndelegations}.IntoProto())
	if err != nil {
		return nil, nil, err
	}

	e.dss.serialisedPending = data
	e.dss.pendingHash = crypto.Hash(data)
	e.dss.pendingChanged = false
	return e.dss.serialisedPending, e.dss.pendingHash, nil
}

// recalculate auto hash/serialise if the state has changed and return the auto serialised form and hash
func (e *Engine) updateAutoAndGet() ([]byte, []byte, error) {
	if !e.dss.autoChanged {
		return e.dss.serialisedAuto, e.dss.autoHash, nil
	}
	auto := e.getAuto()

	data, err := proto.Marshal(types.DelegationAuto{Parties: auto}.IntoProto())
	if err != nil {
		return nil, nil, err
	}

	e.dss.serialisedAuto = data
	e.dss.autoHash = crypto.Hash(data)
	e.dss.autoChanged = false
	return e.dss.serialisedAuto, e.dss.autoHash, nil
}

// get the serialised form and has of the given key
func (e *Engine) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if _, ok := e.keyToSnapshotHandler[k]; !ok {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	return e.keyToSnapshotHandler[k]()
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

func (e *Engine) LoadState(p *types.Payload) error {
	if e.Namespace() != p.Data.Namespace() {
		return ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadDelegationActive:
		return e.restoreActive(pl.DelegationActive)
	case *types.PayloadDelegationPending:
		return e.restorePending(pl.DelegationPending)
	case *types.PayloadDelegationAuto:
		return e.restoreAuto(pl.DelegationAuto)
	default:
		return ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreActive(delegations *types.DelegationActive) error {
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
	e.setActive(entries)
	// after reloading we need to set the dirty flag to true so that we know next time to recalc the hash/serialise
	e.dss.activeChanged = true
	return nil
}

func (e *Engine) restorePending(delegations *types.DelegationPending) error {
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
	e.setPending(entries)
	// after reloading we need to set the dirty flag to true so that we know next time to recalc the hash/serialise
	e.dss.pendingChanged = true
	return nil
}

func (e *Engine) restoreAuto(delegations *types.DelegationAuto) error {
	e.setAuto(delegations.Parties)
	// after reloading we need to set the dirty flag to true so that we know next time to recalc the hash/serialise
	e.dss.autoChanged = true
	return nil
}
