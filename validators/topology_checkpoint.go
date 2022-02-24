package validators

import (
	"context"
	"encoding/base64"
	"sort"
	"time"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

func (e *Topology) Name() types.CheckpointName {
	return types.ValidatorsCheckpoint
}

func (t *Topology) Checkpoint() ([]byte, error) {
	snap := &checkpoint.Validators{
		ValidatorState:      t.getValidatorStateCheckpoint(),
		PendingKeyRotations: t.getCheckpointPendingKeyRotations(),
	}
	return proto.Marshal(snap)
}

func (t *Topology) Load(ctx context.Context, data []byte) error {
	ckp := &checkpoint.Validators{}
	if err := proto.Unmarshal(data, ckp); err != nil {
		return err
	}

	votingPower := make(map[string]int64, len(t.validators))
	for k := range t.validators {
		votingPower[k] = 0
	}

	t.validators = make(map[string]*valState, len(ckp.ValidatorState))
	for _, node := range ckp.ValidatorState {
		t.validators[node.ValidatorUpdate.NodeId] = &valState{
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
			blockAdded:                      int64(t.currentBlockHeight),
			status:                          ValidatorStatus(node.Status),
			statusChangeBlock:               int64(t.currentBlockHeight),
			lastBlockWithPositiveRanking:    int64(t.currentBlockHeight - 1),
			numberOfEthereumEventsForwarded: node.EthEventsForwarded,
			heartbeatTracker: &validatorHeartbeatTracker{
				blockIndex:            0,
				expectedNextHash:      "",
				expectedNexthashSince: time.Time{},
			},
			validatorPower: node.ValidatorPower,
		}
		votingPower[node.ValidatorUpdate.NodeId] = node.ValidatorPower
		t.sendValidatorUpdateEvent(ctx, t.validators[node.ValidatorUpdate.NodeId].data, true)
	}

	for _, pr := range ckp.PendingKeyRotations {
		// skip this key rotation as the node is not parcitipating in the new network
		if _, ok := t.validators[pr.NodeId]; !ok {
			continue
		}

		targetBlockHeight := t.currentBlockHeight + pr.RelativeTargetBlockHeight

		if _, ok := t.pendingPubKeyRotations[targetBlockHeight]; !ok {
			t.pendingPubKeyRotations[targetBlockHeight] = map[string]pendingKeyRotation{}
		}

		t.pendingPubKeyRotations[targetBlockHeight][pr.NodeId] = pendingKeyRotation{
			newPubKey:   pr.NewPubKey,
			newKeyIndex: pr.NewPubKeyIndex,
		}
	}

	nextValidators := make([]string, 0, len(votingPower))
	for k := range votingPower {
		nextValidators = append(nextValidators, k)
	}

	sort.Strings(nextValidators)

	// generate the tendermint updates from the voting power so that in end of the block the validator powers are pushed to tentermint
	vUpdates := make([]tmtypes.ValidatorUpdate, 0, len(nextValidators))
	for _, v := range nextValidators {
		vd := t.validators[v]
		pubkey, err := base64.StdEncoding.DecodeString(vd.data.TmPubKey)
		if err != nil {
			continue
		}
		vd.validatorPower = votingPower[v]
		update := tmtypes.UpdateValidator(pubkey, votingPower[v], "")
		vUpdates = append(vUpdates, update)
	}

	// setting this to true so that at the end of the block
	t.validatorPowerUpdates = vUpdates
	t.newEpochStarted = true
	return nil
}

func (t *Topology) getRelativeBlockHeight(blockHeight, currentBlockHeight uint64) uint64 {
	// this should never happen but (just in case) we want to make sure the key rotation will happen in future
	// so adding it's shifted artificially 2 blocks ahead
	if blockHeight <= currentBlockHeight {
		return 2
	}
	return blockHeight - currentBlockHeight
}

func (t *Topology) getCheckpointPendingKeyRotations() []*checkpoint.PendingKeyRotation {
	rotations := make([]*checkpoint.PendingKeyRotation, 0, len(t.pendingPubKeyRotations)*2)

	blockHeights := make([]uint64, 0, len(t.pendingPubKeyRotations))
	for blockHeight := range t.pendingPubKeyRotations {
		blockHeights = append(blockHeights, blockHeight)
	}
	sort.Slice(blockHeights, func(i, j int) bool { return blockHeights[i] < blockHeights[j] })

	for _, blockHeight := range blockHeights {
		rs := t.pendingPubKeyRotations[blockHeight]
		nodeIDs := make([]string, 0, len(rs))
		for nodeID := range rs {
			nodeIDs = append(nodeIDs, nodeID)
		}
		sort.Strings(nodeIDs)

		for _, nodeID := range nodeIDs {
			r := rs[nodeID]
			rotations = append(rotations, &checkpoint.PendingKeyRotation{
				RelativeTargetBlockHeight: t.getRelativeBlockHeight(blockHeight, t.currentBlockHeight),
				NodeId:                    nodeID,
				NewPubKey:                 r.newPubKey,
				NewPubKeyIndex:            r.newKeyIndex,
			})
		}
	}
	return rotations
}

func (t *Topology) getValidatorStateCheckpoint() []*checkpoint.ValidatorState {
	vsSlice := make([]*checkpoint.ValidatorState, 0, len(t.validators))

	keys := make([]string, 0, len(t.validators))
	for k := range t.validators {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, v := range keys {
		node := t.validators[v]
		vsSlice = append(vsSlice, &checkpoint.ValidatorState{
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
			Status:             int32(node.status),
			EthEventsForwarded: node.numberOfEthereumEventsForwarded,
			ValidatorPower:     node.validatorPower,
		})
	}
	return vsSlice
}
