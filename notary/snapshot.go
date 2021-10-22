package notary

import (
	"context"
	"sort"
	"strings"

	v1 "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

var (
	allKey = (&types.PayloadNotary{}).Key()

	hashKeys = []string{
		allKey,
	}
)

// NewWithSnapshot returns an "extended" Notary type which contains the ability to take engine snapshots.
func NewWithSnapshot(log *logging.Logger, cfg Config, top ValidatorTopology, broker Broker, cmd Commander) *SnapshotNotary {
	log = log.Named(namedLogger)
	return &SnapshotNotary{
		Notary:  New(log, cfg, top, broker, cmd),
		changed: true,
	}
}

type SnapshotNotary struct {
	*Notary

	// snapshot bits
	hash       []byte
	serialised []byte
	changed    bool
}

// StartAggregate is a wrapper to Notary's StartAggregate which also manages the snapshot state.
func (n *SnapshotNotary) StartAggregate(resID string, kind v1.NodeSignatureKind) {
	n.Notary.StartAggregate(resID, kind)
	n.changed = true
}

// AddSig is a wrapper to Notary's AddSig which also manages the snapshot state.
func (n *SnapshotNotary) AddSig(ctx context.Context, pubKey string, ns v1.NodeSignature) ([]v1.NodeSignature, bool, error) {
	sigsout, ok, err := n.Notary.AddSig(ctx, pubKey, ns)
	if err == nil {
		n.changed = true
	}

	return sigsout, ok, err
}

// get the serialised form and hash of the given key.
func (n *SnapshotNotary) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != allKey {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !n.changed {
		return n.serialised, n.hash, nil
	}

	data, err := n.serialiseNotary()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	n.serialised = data
	n.hash = hash
	n.changed = false
	return data, hash, nil
}

func (n *SnapshotNotary) Namespace() types.SnapshotNamespace {
	return types.NotarySnapshot
}

func (n *SnapshotNotary) Keys() []string {
	return hashKeys
}

func (n *SnapshotNotary) GetHash(k string) ([]byte, error) {
	_, hash, err := n.getSerialisedAndHash(k)
	return hash, err
}

func (n *SnapshotNotary) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, _, err := n.getSerialisedAndHash(k)
	return data, nil, err
}

func (n *SnapshotNotary) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if n.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadNotary:
		return nil, n.restoreNotary(pl.Notary)
	default:
		return nil, types.ErrUnknownSnapshotType
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
					Sig:  n.sig,
				},
			)
		}

		// the case where aggregate has started but we have no node sigs
		if len(ns) == 0 {
			sigs = append(sigs, &types.NotarySigs{ID: ik.id, Kind: int32(ik.kind)})
		}
	}

	sort.SliceStable(sigs, func(i, j int) bool {
		switch strings.Compare(sigs[i].ID, sigs[j].ID) {
		case -1:
			return true
		case 1:
			return false
		}

		switch strings.Compare(sigs[i].Node, sigs[j].Node) {
		case -1:
			return true
		case 1:
			return false
		}

		switch strings.Compare(sigs[i].Sig, sigs[j].Sig) {
		case -1:
			return true
		case 1:
			return false
		}

		if sigs[i].Kind == sigs[j].Kind {
			n.log.Panic("could not deterministically order notary sigs for snapshot")
		}

		return sigs[i].Kind < sigs[j].Kind
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

func (n *SnapshotNotary) restoreNotary(notary *types.Notary) error {
	sigs := map[idKind]map[nodeSig]struct{}{}

	for _, s := range notary.Sigs {
		idK := idKind{id: s.ID, kind: v1.NodeSignatureKind(s.Kind)}
		ns := nodeSig{node: s.Node, sig: s.Sig}

		if _, ok := sigs[idK]; !ok {
			sigs[idK] = map[nodeSig]struct{}{}
		}

		if len(ns.node) != 0 && len(ns.sig) != 0 {
			sigs[idK][ns] = struct{}{}
		}
	}

	n.sigs = sigs
	return nil
}
