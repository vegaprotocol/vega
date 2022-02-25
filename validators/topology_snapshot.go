package validators

import (
	"context"
	"sort"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/events"
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

func (t *Topology) Stopped() bool {
	return false
}

func (t *Topology) serialiseNodes() []*snapshot.ValidatorState {
	nodes := make([]*snapshot.ValidatorState, 0, len(t.validators))
	for _, node := range t.validators {
		nodes = append(nodes,
			&snapshot.ValidatorState{
				ValidatorUpdate: &eventspb.ValidatorUpdate{
					NodeId:          node.data.ID,
					VegaPubKey:      node.data.VegaPubKey,
					VegaPubKeyIndex: node.data.VegaPubKeyIndex,
					EthereumAddress: node.data.EthereumAddress,
					TmPubKey:        node.data.TmPubKey,
					InfoUrl:         node.data.InfoURL,
					Country:         node.data.Country,
					Name:            node.data.Name,
					AvatarUrl:       node.data.AvatarURL,
					FromEpoch:       node.data.FromEpoch,
				},
				BlockAdded:                   uint64(node.blockAdded),
				Status:                       int32(node.status),
				StatusChangeBlock:            uint64(node.statusChangeBlock),
				LastBlockWithPositiveRanking: uint64(node.lastBlockWithPositiveRanking),
				EthEventsForwarded:           node.numberOfEthereumEventsForwarded,
				HeartbeatTracker: &snapshot.HeartbeatTracker{
					BlockSigs:             node.heartbeatTracker.blockSigs[:],
					BlockIndex:            int32(node.heartbeatTracker.blockIndex),
					ExpectedNextHash:      node.heartbeatTracker.expectedNextHash,
					ExpectedNextHashSince: node.heartbeatTracker.expectedNexthashSince.UnixNano(),
				},
			},
		)
	}

	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].ValidatorUpdate.NodeId < nodes[j].ValidatorUpdate.NodeId })
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
				NewPubKey:      pr.newPubKey,
				NewPubKeyIndex: pr.newKeyIndex,
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
				ValidatorPerformance:   t.validatorPerformance.Serialize(),
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

func (t *Topology) restorePendingKeyRotations(ctx context.Context, pkrs []*snapshot.PendingKeyRotation) {
	for _, pkr := range pkrs {
		if _, ok := t.pendingPubKeyRotations[pkr.BlockHeight]; !ok {
			t.pendingPubKeyRotations[pkr.BlockHeight] = map[string]pendingKeyRotation{}
		}

		t.pendingPubKeyRotations[pkr.BlockHeight][pkr.NodeId] = pendingKeyRotation{
			newPubKey:   pkr.NewPubKey,
			newKeyIndex: pkr.NewPubKeyIndex,
		}
		data := t.validators[pkr.NodeId]
		t.broker.Send(events.NewKeyRotationEvent(ctx, pkr.NodeId, data.data.VegaPubKey, pkr.NewPubKey, pkr.BlockHeight))
	}
}

func (t *Topology) restore(ctx context.Context, topology *types.Topology) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.validators = map[string]*valState{}

	for _, node := range topology.ValidatorData {
		vs := &valState{
			data: ValidatorData{
				ID:              node.ValidatorUpdate.NodeId,
				VegaPubKey:      node.ValidatorUpdate.VegaPubKey,
				VegaPubKeyIndex: node.ValidatorUpdate.VegaPubKeyIndex,
				EthereumAddress: node.ValidatorUpdate.EthereumAddress,
				TmPubKey:        node.ValidatorUpdate.TmPubKey,
				InfoURL:         node.ValidatorUpdate.InfoUrl,
				Country:         node.ValidatorUpdate.Country,
				Name:            node.ValidatorUpdate.Name,
				AvatarURL:       node.ValidatorUpdate.AvatarUrl,
				FromEpoch:       node.ValidatorUpdate.FromEpoch,
			},
			blockAdded:                      int64(node.BlockAdded),
			status:                          ValidatorStatus(node.Status),
			statusChangeBlock:               int64(node.StatusChangeBlock),
			lastBlockWithPositiveRanking:    int64(node.LastBlockWithPositiveRanking),
			numberOfEthereumEventsForwarded: node.EthEventsForwarded,
			heartbeatTracker: &validatorHeartbeatTracker{
				blockIndex:            int(node.HeartbeatTracker.BlockIndex),
				expectedNextHash:      node.HeartbeatTracker.ExpectedNextHash,
				expectedNexthashSince: time.Unix(0, node.HeartbeatTracker.ExpectedNextHashSince),
			},
		}
		for i := 0; i < 10; i++ {
			vs.heartbeatTracker.blockSigs[i] = node.HeartbeatTracker.BlockSigs[i]
		}
		t.validators[node.ValidatorUpdate.NodeId] = vs

		t.sendValidatorUpdateEvent(ctx, vs.data, true)

		// this node is started and expect to be a validator
		// but so far we haven't seen ourselve as validators for
		// this network.
		if t.isValidatorSetup && !t.isValidator {
			t.checkValidatorDataWithSelfWallets(vs.data)
		}
	}

	t.chainValidators = topology.ChainValidators[:]
	t.restorePendingKeyRotations(ctx, topology.PendingPubKeyRotations)
	t.validatorPerformance.Deserialize(topology.ValidatorPerformance)
	t.tss.changed = true
	return nil
}
