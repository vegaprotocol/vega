package delegation

import (
	"context"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
)

var hashKeys = []string{
	activeKey,
	pendingKey,
	autoKey,
	lastReconKey,
}

type delegationSnapshotState struct {
	changed    map[string]bool
	serialised map[string][]byte
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.DelegationSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

func (e *Engine) serialiseLastReconTime() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadDelegationLastReconTime{
			LastReconcilicationTime: e.lastReconciliation,
		},
	}
	return proto.Marshal(payload.IntoProto())
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
	pending := e.getPendingNew()
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

// get the serialised form and hash of the given key.
func (e *Engine) serialise(k string) ([]byte, error) {
	e.lock.Lock()
	defer e.lock.Unlock()
	if _, ok := e.keyToSerialiser[k]; !ok {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !e.dss.changed[k] {
		return e.dss.serialised[k], nil
	}

	data, err := e.keyToSerialiser[k]()
	if err != nil {
		return nil, err
	}

	e.dss.serialised[k] = data
	e.dss.changed[k] = false
	return data, nil
}

func (e *Engine) HasChanged(k string) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.dss.changed[k]
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadDelegationActive:
		return nil, e.restoreActive(ctx, pl.DelegationActive, p)
	case *types.PayloadDelegationPending:
		return nil, e.restorePending(ctx, pl.DelegationPending, p)
	case *types.PayloadDelegationAuto:
		return nil, e.restoreAuto(pl.DelegationAuto, p)
	case *types.PayloadDelegationLastReconTime:
		return nil, e.restoreLastReconTime(ctx, pl.LastReconcilicationTime, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreLastReconTime(ctx context.Context, t time.Time, p *types.Payload) error {
	var err error
	e.lastReconciliation = t
	e.dss.changed[lastReconKey] = false
	e.dss.serialised[lastReconKey], err = proto.Marshal(p.IntoProto())

	return err
}

func (e *Engine) restoreActive(ctx context.Context, delegations *types.DelegationActive, p *types.Payload) error {
	e.partyDelegationState = map[string]*partyDelegation{}
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
	var err error
	e.dss.changed[activeKey] = false
	e.dss.serialised[activeKey], err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restorePending(ctx context.Context, delegations *types.DelegationPending, p *types.Payload) error {
	e.nextPartyDelegationState = map[string]*partyDelegation{}
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
	e.setPendingNew(ctx, entries)
	var err error
	e.dss.changed[pendingKey] = false
	e.dss.serialised[pendingKey], err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreAuto(delegations *types.DelegationAuto, p *types.Payload) error {
	e.autoDelegationMode = map[string]struct{}{}
	e.setAuto(delegations.Parties)
	var err error
	e.dss.changed[autoKey] = false
	e.dss.serialised[autoKey], err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) onEpochRestore(ctx context.Context, epoch types.Epoch) {
	e.log.Debug("epoch restoration notification received", logging.String("epoch", epoch.String()))
	e.currentEpoch = epoch
}
