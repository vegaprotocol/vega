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

package validators

import (
	"context"
	"encoding/base64"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	tmtypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
)

func (t *Topology) Name() types.CheckpointName {
	return types.ValidatorsCheckpoint
}

func (t *Topology) Checkpoint() ([]byte, error) {
	snap := &checkpoint.Validators{
		ValidatorState:              t.getValidatorStateCheckpoint(),
		PendingKeyRotations:         t.getCheckpointPendingKeyRotations(),
		PendingEthereumKeyRotations: t.getCheckpointPendingEthereumKeyRotations(),
	}
	return proto.Marshal(snap)
}

func (t *Topology) Load(ctx context.Context, data []byte) error {
	ckp := &checkpoint.Validators{}
	if err := proto.Unmarshal(data, ckp); err != nil {
		return err
	}

	toRemove := t.validators
	t.validators = make(map[string]*valState, len(ckp.ValidatorState))
	nextValidators := []string{}
	for _, node := range ckp.ValidatorState {
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
			blockAdded:                      int64(t.currentBlockHeight),
			status:                          ValidatorStatus(node.Status),
			statusChangeBlock:               int64(t.currentBlockHeight),
			lastBlockWithPositiveRanking:    int64(t.currentBlockHeight - 1),
			numberOfEthereumEventsForwarded: node.EthEventsForwarded,
			heartbeatTracker: &validatorHeartbeatTracker{
				blockIndex:            int(node.HeartbeatBlockIndex),
				expectedNextHash:      "",
				expectedNexthashSince: time.Time{},
			},
			validatorPower: node.ValidatorPower,
			rankingScore:   node.RankingScore,
		}

		// we check if its populate so that we remain compatible with old checkpoints
		if len(node.HeartbeatBlockSigs) == 10 {
			for i := 0; i < 10; i++ {
				vs.heartbeatTracker.blockSigs[i] = node.HeartbeatBlockSigs[i]
			}
		}

		// this node is started and expect to be a validator
		// but so far we haven't seen ourselves as validators for
		// this network.
		if t.isValidatorSetup && !t.isValidator {
			t.checkValidatorDataWithSelfWallets(vs.data)
		}

		t.validators[node.ValidatorUpdate.NodeId] = vs
		if t.validators[node.ValidatorUpdate.NodeId].validatorPower > 0 {
			nextValidators = append(nextValidators, node.ValidatorUpdate.NodeId)
		}
		t.sendValidatorUpdateEvent(ctx, t.validators[node.ValidatorUpdate.NodeId].data, true)
		t.checkpointLoaded = true

		delete(toRemove, node.ValidatorUpdate.NodeId)
	}

	// send an update event to remove any validators that were in the genesis file, but not in the checkpoint
	for _, v := range toRemove {
		t.sendValidatorUpdateEvent(ctx, v.data, false)
	}

	t.restoreCheckpointPendingKeyRotations(ckp.PendingKeyRotations)
	t.restoreCheckpointPendingEthereumKeyRotations(ckp.PendingEthereumKeyRotations)

	sort.Strings(nextValidators)

	// generate the tendermint updates from the voting power so that in end of the block the validator powers are pushed to tentermint
	vUpdates := make([]tmtypes.ValidatorUpdate, 0, len(nextValidators))
	for _, v := range nextValidators {
		// NB: if the validator set in the checkpoint doesn't match genesis, vd may be nil
		vd := t.validators[v]
		pubkey, err := base64.StdEncoding.DecodeString(vd.data.TmPubKey)
		if err != nil {
			continue
		}
		update := tmtypes.ValidatorUpdate{
			Power:       vd.validatorPower,
			PubKeyType:  ed25519.KeyType,
			PubKeyBytes: pubkey,
		}
		vUpdates = append(vUpdates, update)
	}

	// setting this to true so we can pass the powers back to tendermint after initChain
	t.validatorPowerUpdates = vUpdates
	t.newEpochStarted = true
	return nil
}

func (t *Topology) restoreCheckpointPendingKeyRotations(rotations []*checkpoint.PendingKeyRotation) {
	for _, pr := range rotations {
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
}

func (t *Topology) restoreCheckpointPendingEthereumKeyRotations(rotations []*checkpoint.PendingEthereumKeyRotation) {
	for _, pr := range rotations {
		// skip this key rotation as the node is not parcitipating in the new network
		if _, ok := t.validators[pr.NodeId]; !ok {
			continue
		}

		targetBlockHeight := t.currentBlockHeight + pr.RelativeTargetBlockHeight

		t.pendingEthKeyRotations.add(targetBlockHeight, PendingEthereumKeyRotation{
			NodeID:     pr.NodeId,
			NewAddress: pr.NewAddress,
		})
	}
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

func (t *Topology) getCheckpointPendingEthereumKeyRotations() []*checkpoint.PendingEthereumKeyRotation {
	outRotations := make([]*checkpoint.PendingEthereumKeyRotation, 0, len(t.pendingEthKeyRotations)*2)

	for blockHeight, rotations := range t.pendingEthKeyRotations {
		for _, r := range rotations {
			outRotations = append(outRotations, &checkpoint.PendingEthereumKeyRotation{
				RelativeTargetBlockHeight: t.getRelativeBlockHeight(blockHeight, t.currentBlockHeight),
				NodeId:                    r.NodeID,
				NewAddress:                r.NewAddress,
			})
		}
	}

	sort.SliceStable(outRotations, func(i, j int) bool {
		if outRotations[i].GetRelativeTargetBlockHeight() == outRotations[j].GetRelativeTargetBlockHeight() {
			return outRotations[i].GetNodeId() < outRotations[j].GetNodeId()
		}
		return outRotations[i].GetRelativeTargetBlockHeight() < outRotations[j].GetRelativeTargetBlockHeight()
	})

	return outRotations
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
			Status:              int32(node.status),
			EthEventsForwarded:  node.numberOfEthereumEventsForwarded,
			ValidatorPower:      node.validatorPower,
			RankingScore:        node.rankingScore,
			HeartbeatBlockIndex: int32(node.heartbeatTracker.blockIndex),
			HeartbeatBlockSigs:  node.heartbeatTracker.blockSigs[:],
		})
	}
	return vsSlice
}
