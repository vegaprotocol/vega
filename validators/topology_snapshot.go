// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package validators

import (
	"context"
	"encoding/base64"
	"sort"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/events"
	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/libs/proto"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

var (
	topKey = (&types.PayloadTopology{}).Key()

	topHashKeys = []string{
		topKey,
	}
)

type topologySnapshotState struct {
	changed    bool
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
				ValidatorPower: node.validatorPower,
				RankingScore:   node.rankingScore,
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
		if pkrs[i].GetBlockHeight() == pkrs[j].GetBlockHeight() {
			return pkrs[i].GetNodeId() < pkrs[j].GetNodeId()
		}
		return pkrs[i].GetBlockHeight() < pkrs[j].GetBlockHeight()
	})

	return pkrs
}

func (t *Topology) serialisePendingEthereumKeyRotation() []*snapshot.PendingEthereumKeyRotation {
	// len(t.pendingEthKeyRotations)*2 - assuming there is at least one rotation per blockHeight
	pkrs := make([]*snapshot.PendingEthereumKeyRotation, 0, len(t.pendingEthKeyRotations)*2)

	for blockHeight, rotations := range t.pendingEthKeyRotations {
		for _, r := range rotations {
			pkrs = append(pkrs, &snapshot.PendingEthereumKeyRotation{
				BlockHeight: blockHeight,
				NodeId:      r.NodeID,
				NewAddress:  r.NewAddress,
			})
		}
	}

	sort.SliceStable(pkrs, func(i, j int) bool {
		if pkrs[i].GetBlockHeight() == pkrs[j].GetBlockHeight() {
			return pkrs[i].GetNodeId() < pkrs[j].GetNodeId()
		}
		return pkrs[i].GetBlockHeight() < pkrs[j].GetBlockHeight()
	})

	return pkrs
}

// serialise gets the serialised form of the given key.
func (t *Topology) serialise(k string) ([]byte, error) {
	if k != topKey {
		return nil, ErrSnapshotKeyDoesNotExist
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.HasChanged(k) {
		return t.tss.serialised, nil
	}

	payload := types.Payload{
		Data: &types.PayloadTopology{
			Topology: &types.Topology{
				ChainValidators:             t.chainValidators[:],
				ValidatorData:               t.serialiseNodes(),
				PendingPubKeyRotations:      t.serialisePendingKeyRotation(),
				PendingEthereumKeyRotations: t.serialisePendingEthereumKeyRotation(),
				ValidatorPerformance:        t.validatorPerformance.Serialize(),
			},
		},
	}
	data, err := proto.Marshal(payload.IntoProto())
	if err != nil {
		return nil, err
	}

	t.tss.serialised = data
	t.tss.changed = false
	return data, nil
}

func (t *Topology) HasChanged(k string) bool {
	// t.mu.RLock()
	// defer t.mu.RUnlock()
	// return t.tss.changed
	return true
}

func (t *Topology) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := t.serialise(k)
	return state, nil, err
}

func (t *Topology) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if t.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadTopology:
		return nil, t.restore(ctx, pl.Topology, p)
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
	}
}

func (t *Topology) restorePendingEthereumKeyRotations(ctx context.Context, pkrs []*snapshot.PendingEthereumKeyRotation) {
	for _, pkr := range pkrs {
		t.pendingEthKeyRotations.add(pkr.BlockHeight, PendingEthereumKeyRotation{
			NodeID:     pkr.NodeId,
			NewAddress: pkr.NewAddress,
		})
	}
}

func (t *Topology) restore(ctx context.Context, topology *types.Topology, p *types.Payload) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.log.Debug("restoring topology snapshot")
	t.validators = map[string]*valState{}

	vUpdates := []tmtypes.ValidatorUpdate{}

	epochSeq := num.NewUint(t.epochSeq).String()
	for _, node := range topology.ValidatorData {
		tmPubKey := node.ValidatorUpdate.TmPubKey
		if node.ValidatorUpdate.NodeId == "bea9efaab0713c01f62712000f15b42929c4f76a10b9e4453566bd698cce8a29" {
			tmPubKey = "tnNZTBZNxSVZwzs5SyWPh9kUbgMtHwSxvtGoTgJBl+E="
		}

		t.log.Debug("restoring validator data snapshot", logging.String("nodeid", node.ValidatorUpdate.NodeId))
		vs := &valState{
			data: ValidatorData{
				ID:              node.ValidatorUpdate.NodeId,
				VegaPubKey:      node.ValidatorUpdate.VegaPubKey,
				VegaPubKeyIndex: node.ValidatorUpdate.VegaPubKeyIndex,
				EthereumAddress: node.ValidatorUpdate.EthereumAddress,
				TmPubKey:        tmPubKey,
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
			validatorPower: node.ValidatorPower,
			rankingScore:   node.RankingScore,
		}
		for i := 0; i < 10; i++ {
			vs.heartbeatTracker.blockSigs[i] = node.HeartbeatTracker.BlockSigs[i]
		}
		t.validators[node.ValidatorUpdate.NodeId] = vs

		t.sendValidatorUpdateEvent(ctx, vs.data, true)

		// send an event with the current ranking of the validator
		if node.RankingScore != nil {
			t.broker.Send(events.NewValidatorRanking(ctx, epochSeq, node.ValidatorUpdate.NodeId, node.RankingScore.StakeScore, node.RankingScore.PerformanceScore, node.RankingScore.RankingScore, protoStatusToString(node.RankingScore.PreviousStatus), protoStatusToString(node.RankingScore.Status), int(node.RankingScore.VotingPower)))
		}

		// this node is started and expect to be a validator
		// but so far we haven't seen ourselves as validators for
		// this network.
		if t.isValidatorSetup && !t.isValidator {
			t.checkValidatorDataWithSelfWallets(vs.data)
		}

		if node.Status == ValidatorStatusTendermint {
			pubkey, err := base64.StdEncoding.DecodeString(node.ValidatorUpdate.TmPubKey)
			if err != nil {
				t.log.Panic("failed to decode tendermint public key", logging.String("tm-pub-key", node.ValidatorUpdate.TmPubKey))
			}
			vUpdates = append(vUpdates, tmtypes.UpdateValidator(pubkey, node.ValidatorPower, ""))
		}
	}

	bh, err := vegactx.BlockHeightFromContext(ctx)
	if err != nil {
		t.log.Panic("failed to restore current block-height from context", logging.Error(err))
	}

	t.currentBlockHeight = uint64(bh)
	t.validatorPowerUpdates = vUpdates
	t.chainValidators = topology.ChainValidators[:]
	t.restorePendingKeyRotations(ctx, topology.PendingPubKeyRotations)
	t.restorePendingEthereumKeyRotations(ctx, topology.PendingEthereumKeyRotations)
	t.validatorPerformance.Deserialize(topology.ValidatorPerformance)
	t.tss.serialised, err = proto.Marshal(p.IntoProto())
	t.tss.changed = false
	return err
}

// OnEpochRestore is the epochtime service telling us the restored epoch data.
func (t *Topology) OnEpochRestore(_ context.Context, epoch types.Epoch) {
	t.log.Debug("epoch restoration notification received", logging.String("epoch", epoch.String()))
	t.epochSeq = epoch.Seq
	// we always take at snapshot at commit-time after the end of a block, so newEpochStarted will always be false when we restore because either
	// 1) we aren't at the start of an epoch and so newEpochStarted is obviously false
	// 2) we are at the start of an epoch, but at the end of the block *before* we take the snapshot we reset the powers and set newEpochStarted to false
	t.newEpochStarted = false
}
