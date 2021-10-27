package validators

import (
	"context"
	"sort"
	"sync"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
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
	mu         sync.Mutex
}

func (tss *topologySnapshotState) setChanged(value bool) {
	tss.mu.Lock()
	tss.changed = value
	tss.mu.Unlock()
}

func (t *Topology) Namespace() types.SnapshotNamespace {
	return types.TopologySnapshot
}

func (t *Topology) Keys() []string {
	return topHashKeys
}

func (t *Topology) serialise() ([]byte, error) {
	nodes := make([]*eventspb.ValidatorUpdate, 0, len(t.validators))

	for _, node := range t.validators {
		nodes = append(nodes,
			&eventspb.ValidatorUpdate{
				NodeId:          node.ID,
				VegaPubKey:      node.VegaPubKey,
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

	payload := types.Payload{
		Data: &types.PayloadTopology{
			Topology: &types.Topology{
				ChainValidators: t.chainValidators[:],
				ValidatorData:   nodes,
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

	t.tss.mu.Lock()
	defer t.tss.mu.Unlock()

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

func (t *Topology) restore(ctx context.Context, topology *types.Topology) error {
	t.tss.mu.Lock()
	defer t.tss.mu.Unlock()

	walletID := t.wallet.ID().Hex()

	for _, node := range topology.ValidatorData {
		t.validators[node.NodeId] = ValidatorData{
			ID:              node.NodeId,
			VegaPubKey:      node.VegaPubKey,
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
	t.tss.changed = true
	return nil
}
