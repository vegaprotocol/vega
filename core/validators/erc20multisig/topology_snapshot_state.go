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

package erc20multisig

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	verifiedStateKey = (&types.PayloadERC20MultiSigTopologyVerified{}).Key()
	pendingStateKey  = (&types.PayloadERC20MultiSigTopologyPending{}).Key()

	hashKeys = []string{
		verifiedStateKey,
		pendingStateKey,
	}
)

type topologySnapshotState struct {
	serialisedVerifiedState []byte
	serialisedPendingState  []byte
}

func (t *Topology) Namespace() types.SnapshotNamespace {
	return types.ERC20MultiSigTopologySnapshot
}

func (t *Topology) Keys() []string {
	return hashKeys
}

func (t *Topology) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := t.serialise(k)
	return data, nil, err
}

func (t *Topology) Stopped() bool {
	return false
}

func (t *Topology) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if t.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadERC20MultiSigTopologyVerified:
		return nil, t.restoreVerifiedState(ctx, pl.Verified, payload)
	case *types.PayloadERC20MultiSigTopologyPending:
		return nil, t.restorePendingState(ctx, pl.Pending, payload)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (t *Topology) restoreVerifiedState(
	_ context.Context, s *snapshotpb.ERC20MultiSigTopologyVerified, p *types.Payload,
) error {
	t.log.Debug("restoring snapshot verified state")
	if s.Threshold != nil {
		t.log.Debug("restoring threshold")
		t.threshold = types.SignerThresholdSetEventFromEventProto(s.Threshold)
	}

	t.log.Debug("restoring seen events", logging.Int("n", len(s.SeenEvents)))
	for _, v := range s.SeenEvents {
		t.seen[v] = struct{}{}
	}

	t.log.Debug("restoring signers", logging.Int("n", len(s.Signers)))
	for _, v := range s.Signers {
		t.signers[v] = struct{}{}
	}

	t.log.Debug("restoring events per address", logging.Int("n", len(s.EventsPerAddress)))
	for _, v := range s.EventsPerAddress {
		events := make([]*types.SignerEvent, 0, len(v.Events))
		for _, e := range v.Events {
			events = append(events, types.SignerEventFromEventProto(e))
		}
		t.eventsPerAddress[v.Address] = events
	}

	var err error
	t.tss.serialisedVerifiedState, err = proto.Marshal(p.IntoProto())
	return err
}

func (t *Topology) restorePendingState(
	_ context.Context, s *snapshotpb.ERC20MultiSigTopologyPending, p *types.Payload,
) error {
	t.log.Debug("restoring snapshot pending state")
	t.log.Debug("restoring witness signers", logging.Int("n", len(s.WitnessedSigners)))
	for _, v := range s.WitnessedSigners {
		t.witnessedSigners[v] = struct{}{}
	}

	t.log.Debug("restoring witness threshold sets", logging.Int("n", len(s.WitnessedThresholdSets)))
	for _, v := range s.WitnessedThresholdSets {
		t.witnessedThresholds[v] = struct{}{}
	}

	t.log.Debug("restoring pending signers", logging.Int("n", len(s.PendingSigners)))
	for _, v := range s.PendingSigners {
		evt := types.SignerEventFromEventProto(v)
		pending := &pendingSigner{
			SignerEvent: evt,
			check:       func() error { return t.ocv.CheckSignerEvent(evt) },
		}

		t.pendingSigners[evt.ID] = pending
		// if we have witnessed it already,
		if _, ok := t.witnessedSigners[evt.ID]; !ok {
			if err := t.witness.RestoreResource(pending, t.onEventVerified); err != nil {
				t.log.Panic("unable to restore pending signer resource", logging.String("id", pending.ID), logging.Error(err))
			}
		}
	}

	t.log.Debug("restoring pending threshold set", logging.Int("n", len(s.PendingThresholdSet)))
	for _, v := range s.PendingThresholdSet {
		evt := types.SignerThresholdSetEventFromEventProto(v)
		pending := &pendingThresholdSet{
			SignerThresholdSetEvent: evt,
			check:                   func() error { return t.ocv.CheckThresholdSetEvent(evt) },
		}

		t.pendingThresholds[evt.ID] = pending
		// if we have witnessed it already,
		if _, ok := t.witnessedThresholds[evt.ID]; !ok {
			if err := t.witness.RestoreResource(pending, t.onEventVerified); err != nil {
				t.log.Panic("unable to restore pending threshold resource", logging.String("id", pending.ID), logging.Error(err))
			}
		}
	}

	var err error
	t.tss.serialisedPendingState, err = proto.Marshal(p.IntoProto())
	return err
}

func (t *Topology) serialiseVerifiedState() ([]byte, error) {
	out := &snapshotpb.ERC20MultiSigTopologyVerified{}
	t.log.Debug("serialising snapshot verified state")
	// first serialise seen events
	t.log.Debug("serialising seen", logging.Int("n", len(t.seen)))
	out.SeenEvents = make([]string, 0, len(t.seen))
	for k := range t.seen {
		out.SeenEvents = append(out.SeenEvents, k)
	}
	sort.Strings(out.SeenEvents)

	// then the current known list of signers
	t.log.Debug("serialising signers", logging.Int("n", len(t.signers)))
	out.Signers = make([]string, 0, len(t.signers))
	for k := range t.signers {
		out.Signers = append(out.Signers, k)
	}
	// sort it + reuse it next in the eventsPerAddress
	sort.Strings(out.Signers)

	evts := make([]*snapshotpb.SignerEventsPerAddress, 0, len(t.eventsPerAddress))
	// now the signers events
	for k, v := range t.eventsPerAddress {
		events := make([]*eventspb.ERC20MultiSigSignerEvent, 0, len(v))

		t.log.Debug("serialising events", logging.String("signer", k), logging.Int("n", len(v)))
		for _, v := range v {
			events = append(events, v.IntoProto())
		}

		evts = append(
			evts,
			&snapshotpb.SignerEventsPerAddress{
				Address: k,
				Events:  events,
			},
		)
	}
	sort.SliceStable(evts, func(i, j int) bool { return evts[i].Address < evts[j].Address })
	out.EventsPerAddress = evts

	// finally do the current threshold
	if t.threshold != nil {
		t.log.Debug("serialising threshold")
		out.Threshold = t.threshold.IntoProto()
	}

	return proto.Marshal(types.Payload{
		Data: &types.PayloadERC20MultiSigTopologyVerified{
			Verified: out,
		},
	}.IntoProto())
}

func (t *Topology) serialisePendingState() ([]byte, error) {
	t.log.Debug("serialising pending state")
	out := &snapshotpb.ERC20MultiSigTopologyPending{}

	t.log.Debug("serialising witness signers", logging.Int("n", len(t.witnessedSigners)))
	out.WitnessedSigners = make([]string, 0, len(t.witnessedSigners))
	for k := range t.witnessedSigners {
		out.WitnessedSigners = append(out.WitnessedSigners, k)
	}
	sort.Strings(out.WitnessedSigners)

	t.log.Debug("serialising witness threshold sets", logging.Int("n", len(t.witnessedThresholds)))
	out.WitnessedThresholdSets = make([]string, 0, len(t.witnessedThresholds))
	for k := range t.witnessedThresholds {
		out.WitnessedThresholdSets = append(out.WitnessedThresholdSets, k)
	}
	sort.Strings(out.WitnessedThresholdSets)

	t.log.Debug("serialising pending signers", logging.Int("n", len(t.pendingSigners)))
	out.PendingSigners = make([]*eventspb.ERC20MultiSigSignerEvent, 0, len(t.pendingSigners))
	for _, v := range t.pendingSigners {
		out.PendingSigners = append(out.PendingSigners, v.IntoProto())
	}
	sort.SliceStable(out.PendingSigners, func(i, j int) bool {
		return out.PendingSigners[i].Id < out.PendingSigners[j].Id
	})

	t.log.Debug("serialising pending thresholds", logging.Int("n", len(t.pendingThresholds)))
	out.PendingThresholdSet = make([]*eventspb.ERC20MultiSigThresholdSetEvent, 0, len(t.pendingThresholds))
	for _, v := range t.pendingThresholds {
		out.PendingThresholdSet = append(out.PendingThresholdSet, v.IntoProto())
	}
	sort.SliceStable(out.PendingThresholdSet, func(i, j int) bool {
		return out.PendingThresholdSet[i].Id < out.PendingThresholdSet[j].Id
	})

	return proto.Marshal(types.Payload{
		Data: &types.PayloadERC20MultiSigTopologyPending{
			Pending: out,
		},
	}.IntoProto())
}

func (t *Topology) serialiseK(serialFunc func() ([]byte, error), dataField *[]byte) ([]byte, error) {
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	return data, nil
}

// get the serialised form of the given key.
func (t *Topology) serialise(k string) ([]byte, error) {
	switch k {
	case verifiedStateKey:
		return t.serialiseK(t.serialiseVerifiedState, &t.tss.serialisedVerifiedState)
	case pendingStateKey:
		return t.serialiseK(t.serialisePendingState, &t.tss.serialisedPendingState)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (t *Topology) OnStateLoaded(ctx context.Context) error {
	// tell the internal EEF where it got up to so we do not resend events we're already seen
	lastSeen := t.getLastBlockSeen()
	if lastSeen != 0 {
		t.log.Info("restoring multisig starting block", logging.Uint64("block", lastSeen))
		t.ethEventSource.UpdateMultisigControlStartingBlock(t.getLastBlockSeen())
	}
	return nil
}
