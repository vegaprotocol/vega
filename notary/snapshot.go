package notary

import (
	"context"
	"encoding/hex"
	"sort"
	"strings"

	v1 "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	allKey = (&types.PayloadNotary{}).Key()

	hashKeys = []string{
		allKey,
	}
)

// NewWithSnapshot returns an "extended" Notary type which contains the ability to take engine snapshots.
func NewWithSnapshot(
	log *logging.Logger,
	cfg Config,
	top ValidatorTopology,
	broker Broker,
	cmd Commander,
	tt TimeTicker,
) *SnapshotNotary {
	log = log.Named(namedLogger)
	return &SnapshotNotary{
		Notary:  New(log, cfg, top, broker, cmd, tt),
		changed: true,
	}
}

type SnapshotNotary struct {
	*Notary

	// snapshot bits
	serialised []byte
	changed    bool
}

// StartAggregate is a wrapper to Notary's StartAggregate which also manages the snapshot state.
func (n *SnapshotNotary) StartAggregate(
	resource string,
	kind v1.NodeSignatureKind,
	signature []byte,
) {
	n.Notary.StartAggregate(resource, kind, signature)
	n.changed = true
}

// RegisterSignature is a wrapper to Notary's RegisterSignature which also manages the snapshot state.
func (n *SnapshotNotary) RegisterSignature(
	ctx context.Context,
	pubKey string,
	ns v1.NodeSignature,
) error {
	err := n.Notary.RegisterSignature(ctx, pubKey, ns)
	if err == nil {
		n.changed = true
	}

	return err
}

// get the serialised form of the given key.
func (n *SnapshotNotary) serialise(k string) ([]byte, error) {
	if k != allKey {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !n.HasChanged(k) {
		return n.serialised, nil
	}

	data, err := n.serialiseNotary()
	if err != nil {
		return nil, err
	}

	n.serialised = data
	n.changed = false
	return data, nil
}

func (n *SnapshotNotary) Namespace() types.SnapshotNamespace {
	return types.NotarySnapshot
}

func (n *SnapshotNotary) Keys() []string {
	return hashKeys
}

func (n *SnapshotNotary) Stopped() bool {
	return false
}

func (n *SnapshotNotary) HasChanged(k string) bool {
	return true
	// return n.changed
}

func (n *SnapshotNotary) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := n.serialise(k)
	return data, nil, err
}

func (n *SnapshotNotary) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if n.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadNotary:
		return nil, n.restoreNotary(pl.Notary, payload)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (n *SnapshotNotary) OfferSignatures(
	kind types.NodeSignatureKind,
	// a callback taking a list of resource that a signature is required
	// for, returning a map of signature for given resources
	f func(resource string) []byte,
) {
	for k, v := range n.retries.txs {
		if k.kind != kind {
			continue
		}
		if signature := f(k.id); signature != nil {
			v.signature = signature
		}
	}
}

// serialiseLimits returns the engine's limit data as marshalled bytes.
func (n *SnapshotNotary) serialiseNotary() ([]byte, error) {
	sigs := make([]*types.NotarySigs, 0, len(n.sigs)) // it will likely be longer than this but we don't know yet
	for ik, ns := range n.sigs {
		for n := range ns {
			sigs = append(sigs,
				&types.NotarySigs{
					ID:   ik.id,
					Kind: int32(ik.kind),
					Node: n.node,
					Sig:  hex.EncodeToString([]byte(n.sig)),
				},
			)
		}

		// the case where aggregate has started but we have no node sigs
		if len(ns) == 0 {
			sigs = append(sigs, &types.NotarySigs{ID: ik.id, Kind: int32(ik.kind)})
		}
	}

	sort.SliceStable(sigs, func(i, j int) bool {
		// sigs could be "" so we need to sort on ID as well
		switch strings.Compare(sigs[i].ID, sigs[j].ID) {
		case -1:
			return true
		case 1:
			return false
		}

		return sigs[i].Sig < sigs[j].Sig
	})

	pl := types.Payload{
		Data: &types.PayloadNotary{
			Notary: &types.Notary{
				Sigs: sigs,
			},
		},
	}
	return proto.Marshal(pl.IntoProto())
}

func (n *SnapshotNotary) restoreNotary(notary *types.Notary, p *types.Payload) error {
	var (
		sigs    = map[idKind]map[nodeSig]struct{}{}
		retries = &txTracker{
			txs: map[idKind]*signatureTime{},
		}
		isValidator = n.Notary.top.IsValidator()
		selfSigned  = map[idKind]bool{}
		self        = n.Notary.top.SelfVegaPubKey()
	)
	for _, s := range notary.Sigs {
		idK := idKind{id: s.ID, kind: v1.NodeSignatureKind(s.Kind)}

		sig, err := hex.DecodeString(s.Sig)
		if err != nil {
			n.log.Panic("invalid signature from snapshot", logging.Error(err))
		}
		ns := nodeSig{node: s.Node, sig: string(sig)}

		if isValidator {
			signed := selfSigned[idK]
			if !signed {
				selfSigned[idK] = strings.EqualFold(s.Node, self)
			}
		}

		if _, ok := sigs[idK]; !ok {
			sigs[idK] = map[nodeSig]struct{}{}
		}

		if len(ns.node) != 0 && len(ns.sig) != 0 {
			sigs[idK][ns] = struct{}{}
		}
	}

	for resource, ok := range selfSigned {
		if !ok {
			// this is not signed, just add it to the retries list
			retries.Add(resource, nil)
		}
	}

	n.sigs = sigs
	n.retries = retries
	var err error
	n.changed = false
	n.serialised, err = proto.Marshal(p.IntoProto())
	return err
}
