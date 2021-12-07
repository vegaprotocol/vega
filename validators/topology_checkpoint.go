package validators

import (
	"context"
	"fmt"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

func (e *Topology) Name() types.CheckpointName {
	return types.KeyRotationsCheckpoint
}

func (t *Topology) Checkpoint() ([]byte, error) {
	if len(t.pendingPubKeyRotations) == 0 {
		return nil, nil
	}
	snap := &checkpoint.KeyRotations{
		PendingKeyRotations: t.getCheckpointPendingKeyRotations(),
	}
	fmt.Println("snap:", snap)
	return proto.Marshal(snap)
}

func (t *Topology) Load(_ context.Context, data []byte) error {
	ckp := &checkpoint.KeyRotations{}
	if err := proto.Unmarshal(data, ckp); err != nil {
		return err
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
	for blockHeight, rs := range t.pendingPubKeyRotations {
		for nodeID, r := range rs {
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
