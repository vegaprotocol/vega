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

package delegation

import (
	"context"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
)

var hashKeys = []string{
	activeKey,
	pendingKey,
	autoKey,
	lastReconKey,
}

type delegationSnapshotState struct {
	serialisedActive    []byte
	serialisedPending   []byte
	serialisedAuto      []byte
	serialisedLastRecon []byte
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

func (e *Engine) serialiseK(serialFunc func() ([]byte, error), dataField *[]byte) ([]byte, error) {
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	return data, nil
}

// get the serialised form and hash of the given key.
func (e *Engine) serialise(k string) ([]byte, error) {
	switch k {
	case activeKey:
		return e.serialiseK(e.serialiseActive, &e.dss.serialisedActive)
	case pendingKey:
		return e.serialiseK(e.serialisePending, &e.dss.serialisedPending)
	case autoKey:
		return e.serialiseK(e.serialiseAuto, &e.dss.serialisedAuto)
	case lastReconKey:
		return e.serialiseK(e.serialiseLastReconTime, &e.dss.serialisedLastRecon)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
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
		return nil, e.restoreLastReconTime(pl.LastReconcilicationTime, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreLastReconTime(t time.Time, p *types.Payload) error {
	var err error
	e.lastReconciliation = t
	e.dss.serialisedLastRecon, err = proto.Marshal(p.IntoProto())

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
	e.dss.serialisedActive, err = proto.Marshal(p.IntoProto())
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
	e.dss.serialisedPending, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) restoreAuto(delegations *types.DelegationAuto, p *types.Payload) error {
	e.autoDelegationMode = map[string]struct{}{}
	e.setAuto(delegations.Parties)
	var err error
	e.dss.serialisedAuto, err = proto.Marshal(p.IntoProto())
	return err
}

func (e *Engine) onEpochRestore(ctx context.Context, epoch types.Epoch) {
	e.log.Debug("epoch restoration notification received", logging.String("epoch", epoch.String()))
	e.currentEpoch = epoch
}
