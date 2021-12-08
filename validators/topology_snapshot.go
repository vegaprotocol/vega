package validators

import (
	"context"
	"sort"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

var (
	topKey = (&types.PayloadTopology{}).Key()

	topHashKeys = []string{
		topKey,
	}
)

type topologySnapshotState struct {
	changed    bool
	hash       []byte
	serialised []byte
}

func (t *Topology) Namespace() types.SnapshotNamespace {
	return types.TopologySnapshot
}

func (t *Topology) Keys() []string {
	return topHashKeys
}

func (t *Topology) serialiseNodes() []*eventspb.ValidatorUpdate {
	nodes := make([]*eventspb.ValidatorUpdate, 0, len(t.validators))

	for _, node := range t.validators {
		nodes = append(nodes,
			&eventspb.ValidatorUpdate{
				NodeId:          node.ID,
				VegaPubKey:      node.VegaPubKey,
				VegaPubKeyIndex: node.VegaPubKeyIndex,
				EthereumAddress: node.EthereumAddress,
				TmPubKey:        node.TmPubKey,
				InfoUrl:         node.InfoURL,
				Country:         node.Country,
				Name:            node.Name,
				AvatarUrl:       node.AvatarURL,
			},
		)
	}

	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].NodeId < nodes[j].NodeId })
	return nodes
}

func (t *Topology) serialisePendingKeyRotation() []*snapshot.PendingKeyRotation {
	// len(t.pendingPubKeyRotations)*2 - assuming there is at least one rotation per blockHeight
	pkrs := make([]*snapshot.PendingKeyRotation, 0, len(t.pendingPubKeyRotations)*2)

	for blockHeight, rotations := range t.pendingPubKeyRotations {
		for nodeID, pr := range rotations {
			pkrs = append(pkrs, &snapshot.PendingKeyRotation{
				BlockHeight:    blockHeight,
				NodeId:         nodeID,
				NewPubKey:      pr.NewPubKey,
				NewPubKeyIndex: pr.NewKeyIndex,
			})
		}
	}

	sort.SliceStable(pkrs, func(i, j int) bool {
		if pkrs[i].BlockHeight == pkrs[j].BlockHeight {
			return pkrs[i].NodeId < pkrs[j].NodeId
		}
		return pkrs[i].BlockHeight < pkrs[j].BlockHeight
	})

	return pkrs
}

func (t *Topology) serialise() ([]byte, error) {
	payload := types.Payload{
		Data: &types.PayloadTopology{
			Topology: &types.Topology{
				ChainValidators:        t.chainValidators[:],
				ValidatorData:          t.serialiseNodes(),
				PendingPubKeyRotations: t.serialisePendingKeyRotation(),
			},
		},
	}
	x := payload.IntoProto()
	return proto.Marshal(x)
}

// get the serialised form and hash of the given key.
func (t *Topology) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if k != topKey {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.tss.changed {
		return t.tss.serialised, t.tss.hash, nil
	}

	data, err := t.serialise()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	t.tss.serialised = data
	t.tss.hash = hash
	t.tss.changed = false
	return data, hash, nil
}

func (t *Topology) GetHash(k string) ([]byte, error) {
	_, hash, err := t.getSerialisedAndHash(k)
	return hash, err
}

func (t *Topology) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, _, err := t.getSerialisedAndHash(k)
	return state, nil, err
}

func (t *Topology) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if t.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadTopology:
		return nil, t.restore(ctx, pl.Topology)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (t *Topology) restorePendingKeyRotations(pkrs []*snapshot.PendingKeyRotation) {
	for _, pkr := range pkrs {
		if _, ok := t.pendingPubKeyRotations[pkr.BlockHeight]; !ok {
			t.pendingPubKeyRotations[pkr.BlockHeight] = map[string]PendingKeyRotation{}
		}

		t.pendingPubKeyRotations[pkr.BlockHeight][pkr.NodeId] = PendingKeyRotation{
			NewPubKey:   pkr.NewPubKey,
			NewKeyIndex: pkr.NewPubKeyIndex,
		}
	}
}

func (t *Topology) restore(ctx context.Context, topology *types.Topology) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	walletID := t.wallet.ID().Hex()

	for _, node := range topology.ValidatorData {
		t.validators[node.NodeId] = ValidatorData{
			ID:              node.NodeId,
			VegaPubKey:      node.VegaPubKey,
			VegaPubKeyIndex: node.VegaPubKeyIndex,
			EthereumAddress: node.EthereumAddress,
			TmPubKey:        node.TmPubKey,
			InfoURL:         node.InfoUrl,
			Country:         node.Country,
			Name:            node.Name,
			AvatarURL:       node.AvatarUrl,
		}

		if walletID == node.NodeId {
			t.isValidator = true
		}
	}
	t.chainValidators = topology.ChainValidators[:]
	t.restorePendingKeyRotations(topology.PendingPubKeyRotations)
	t.tss.changed = true
	return nil
}
