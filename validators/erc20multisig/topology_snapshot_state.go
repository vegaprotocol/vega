package erc20multisig

import (
	"context"
	"sort"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	snapshotpb "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"google.golang.org/protobuf/proto"
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
	hash       map[string][]byte
	serialised map[string][]byte
	changed    map[string]bool
}

func (s *Topology) Namespace() types.SnapshotNamespace {
	return types.ERC20MultiSigTopologySnapshot
}

func (s *Topology) Keys() []string {
	return hashKeys
}

func (s *Topology) GetHash(k string) ([]byte, error) {
	_, hash, err := s.getSerialisedAndHash(k)
	return hash, err
}

func (s *Topology) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, _, err := s.getSerialisedAndHash(k)
	return data, nil, err
}

func (s *Topology) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadERC20MultiSigTopologyVerified:
		return nil, s.restoreVerifiedState(ctx, pl.Verified)
	case *types.PayloadERC20MultiSigTopologyPending:
		return nil, s.restorePendingState(ctx, pl.Pending)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (t *Topology) restoreVerifiedState(
	_ context.Context, s *snapshotpb.ERC20MultiSigTopologyVerified) error {
	t.threshold = types.SignerThresholdSetEventFromEventProto(s.Threshold)
	for _, v := range s.SeenEvents {
		t.seen[v] = struct{}{}
	}
	for _, v := range s.Signers {
		t.signers[v] = struct{}{}
	}
	for _, v := range s.EventsPerAddress {
		events := make([]*types.SignerEvent, 0, len(v.Events))
		for _, e := range v.Events {
			events = append(events, types.SignerEventFromEventProto(e))
		}
		t.eventsPerAddress[v.Address] = events
	}

	return nil
}

func (t *Topology) restorePendingState(
	_ context.Context, s *snapshotpb.ERC20MultiSigTopologyPending) error {
	for _, v := range s.WitnessedSigners {
		t.witnessedSigners[v] = struct{}{}
	}
	for _, v := range s.WitnessedThresholdSets {
		t.witnessedThresholds[v] = struct{}{}
	}
	for _, v := range s.PendingSigners {
		evt := types.SignerEventFromEventProto(v)
		pending := &pendingSigner{
			SignerEvent: evt,
			check:       func() error { return t.ocv.CheckSignerEvent(evt) },
		}

		t.pendingSigners[evt.ID] = pending
		// if we have witnessed it already,
		if _, ok := t.witnessedSigners[evt.ID]; !ok {
			t.witness.RestoreResource(pending, t.onEventVerified)
		}
	}
	for _, v := range s.PendingThresholdSet {
		evt := types.SignerThresholdSetEventFromEventProto(v)
		pending := &pendingThresholdSet{
			SignerThresholdSetEvent: evt,
			check:                   func() error { return t.ocv.CheckThresholdSetEvent(evt) },
		}

		t.pendingThresholds[evt.ID] = pending
		// if we have witnessed it already,
		if _, ok := t.witnessedThresholds[evt.ID]; !ok {
			t.witness.RestoreResource(pending, t.onEventVerified)
		}
	}

	return nil
}

func (t *Topology) serialiseVerifiedState() ([]byte, error) {
	out := &snapshotpb.ERC20MultiSigTopologyVerified{}

	// first serialise seen events
	out.SeenEvents = make([]string, 0, len(t.seen))
	for k := range t.seen {
		out.SeenEvents = append(out.SeenEvents, k)
	}
	sort.Strings(out.SeenEvents)

	// then the current known list of signers
	out.Signers = make([]string, 0, len(t.signers))
	for k := range t.signers {
		out.Signers = append(out.Signers, k)
	}
	// sort it + reuse it next in the eventsPerAddress
	sort.Strings(out.Signers)

	out.EventsPerAddress = make([]*snapshotpb.SignerEventsPerAddress, 0, len(t.eventsPerAddress))
	// now the signers events
	for _, v := range out.Signers {
		addressEvents := t.eventsPerAddress[v]
		events := make([]*eventspb.ERC20MultiSigSignerEvent, 0, len(addressEvents))

		for _, v := range addressEvents {
			events = append(events, v.IntoProto())
		}

		out.EventsPerAddress = append(
			out.EventsPerAddress,
			&snapshotpb.SignerEventsPerAddress{
				Address: v,
				Events:  events,
			},
		)
	}

	// finally do the current threshold
	out.Threshold = t.threshold.IntoProto()

	return proto.Marshal(out)
}

func (t *Topology) serialisePendingState() ([]byte, error) {
	out := &snapshotpb.ERC20MultiSigTopologyPending{}
	out.WitnessedSigners = make([]string, 0, len(t.witnessedSigners))
	for k := range t.witnessedSigners {
		out.WitnessedSigners = append(out.WitnessedSigners, k)
	}
	sort.Strings(out.WitnessedSigners)

	out.WitnessedThresholdSets = make([]string, 0, len(t.witnessedThresholds))
	for k := range t.witnessedSigners {
		out.WitnessedThresholdSets = append(out.WitnessedThresholdSets, k)
	}
	sort.Strings(out.WitnessedThresholdSets)

	out.PendingSigners = make([]*eventspb.ERC20MultiSigSignerEvent, 0, len(t.pendingSigners))
	for _, v := range t.pendingSigners {
		out.PendingSigners = append(out.PendingSigners, v.IntoProto())
	}
	sort.SliceStable(out.PendingSigners, func(i, j int) bool {
		return out.PendingSigners[i].Id < out.PendingSigners[j].Id
	})

	out.PendingThresholdSet = make([]*eventspb.ERC20MultiSigThresholdSetEvent, 0, len(t.pendingThresholds))
	for _, v := range t.pendingThresholds {
		out.PendingThresholdSet = append(out.PendingThresholdSet, v.IntoProto())
	}
	sort.SliceStable(out.PendingThresholdSet, func(i, j int) bool {
		return out.PendingThresholdSet[i].Id < out.PendingThresholdSet[j].Id
	})

	return proto.Marshal(out)
}

// get the serialised form and hash of the given key.
func (t *Topology) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if _, ok := t.keyToSerialiser[k]; !ok {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !t.tss.changed[k] {
		return t.tss.serialised[k], t.tss.hash[k], nil
	}

	data, err := t.keyToSerialiser[k]()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	t.tss.serialised[k] = data
	t.tss.hash[k] = hash
	t.tss.changed[k] = false
	return data, hash, nil
}
