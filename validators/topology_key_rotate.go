package validators

import (
	"context"
	"errors"
	"fmt"
	"sort"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrTargetBlockHeightMustBeGraterThanCurrentHeight          = errors.New("target block height must be greater then current block")
	ErrNewVegaPubKeyNumberMustBeGreaterThenCurrentPubKeyNumber = errors.New("a new vega public key number must be greather then current public key number")
	ErrInvalidVegaPubKeyForNode                                = errors.New("current vega public key is invalid for node")
	ErrNodeAlreadyHasPendingKeyRotation                        = errors.New("node already has a pending key rotation")
	ErrCurrentPubKeyHashDoesNotMatch                           = errors.New("current public key hash does not match")
)

type pendingKeyRotation struct {
	newPubKey    string
	newKeyNumber uint32
}

// pendingKeyRotationMapping maps a block height => node id => new pending key rotation.
type pendingKeyRotationMapping map[uint64]map[string]pendingKeyRotation

func (pr pendingKeyRotationMapping) getSortedNodeIDsPerHeight(height uint64) []string {
	rotationsPerHeight := pr[height]
	if len(rotationsPerHeight) == 0 {
		return nil
	}

	nodeIDs := make([]string, 0, len(rotationsPerHeight))
	for nodeID := range rotationsPerHeight {
		nodeIDs = append(nodeIDs, nodeID)
	}

	sort.Strings(nodeIDs)

	return nodeIDs
}

type KeyRotation struct {
	NodeID      string
	OldPubKey   string
	NewPubKey   string
	BlockHeight uint64
}

// processedKeyRotationMapping maps node id => slice of key rotations.
type processedKeyRotationMapping map[string][]KeyRotation

func (t *Topology) hasPendingKeyRotation(nodeID string) bool {
	for _, rotationsPerNodeID := range t.pendingPubKeyRotations {
		if _, ok := rotationsPerNodeID[nodeID]; ok {
			return true
		}
	}
	return false
}

func (t *Topology) AddKeyRotate(ctx context.Context, nodeID string, currentBlockHeight uint64, kr *commandspb.KeyRotateSubmission) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	node, ok := t.validators[nodeID]
	if !ok {
		return fmt.Errorf("failed to add key rotate for non existing node %q", nodeID)
	}

	if t.hasPendingKeyRotation(nodeID) {
		return ErrNodeAlreadyHasPendingKeyRotation
	}

	if currentBlockHeight > kr.TargetBlock {
		return ErrTargetBlockHeightMustBeGraterThanCurrentHeight
	}

	if node.VegaPubKeyNumber >= kr.KeyNumber {
		return ErrNewVegaPubKeyNumberMustBeGreaterThenCurrentPubKeyNumber
	}

	if node.HashVegaPubKey() != kr.CurrentPubKeyHash {
		return ErrCurrentPubKeyHashDoesNotMatch
	}

	if _, ok = t.pendingPubKeyRotations[kr.TargetBlock]; !ok {
		t.pendingPubKeyRotations[kr.TargetBlock] = map[string]pendingKeyRotation{}
	}
	t.pendingPubKeyRotations[kr.TargetBlock][nodeID] = pendingKeyRotation{
		newPubKey:    kr.NewPubKey,
		newKeyNumber: kr.KeyNumber,
	}

	t.tss.changed = true

	return nil
}

func (t *Topology) NotifyOnKeyChange(fns ...func(ctx context.Context, oldPubKey, newPubKey string)) {
	t.pubKeyChangeListeners = append(t.pubKeyChangeListeners, fns...)
}

func (t *Topology) notifyKeyChange(ctx context.Context, oldPubKey, newPubKey string) {
	for _, f := range t.pubKeyChangeListeners {
		f(ctx, oldPubKey, newPubKey)
	}
}

func (t *Topology) addProcessedKeyRotation(nodeID, oldPubKey, newPubKey string, blockHeight uint64) {
	if _, ok := t.processedPubKeyRotations[nodeID]; !ok {
		t.processedPubKeyRotations[nodeID] = []KeyRotation{}
	}

	t.processedPubKeyRotations[nodeID] = append(t.processedPubKeyRotations[nodeID], KeyRotation{
		NodeID:      nodeID,
		OldPubKey:   oldPubKey,
		NewPubKey:   newPubKey,
		BlockHeight: blockHeight,
	})
}

// GetKeyRotations returns a history of all processed key rotations per given node.
func (t *Topology) GetKeyRotations(nodeID string) []KeyRotation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.processedPubKeyRotations[nodeID]
}

func (t *Topology) BeginBlock(ctx context.Context, blockHeight uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// key swaps should run in deterministic order
	nodeIDs := t.pendingPubKeyRotations.getSortedNodeIDsPerHeight(blockHeight)
	if len(nodeIDs) == 0 {
		return
	}

	for _, nodeID := range nodeIDs {
		data, ok := t.validators[nodeID]
		if !ok {
			// this should never happened, but just to be safe
			t.log.Error("failed to rotate key due to non existing validator", logging.String("nodeID", nodeID))
			continue
		}

		oldPubKey := data.VegaPubKey
		rotation := t.pendingPubKeyRotations[blockHeight][nodeID]

		data.VegaPubKey = rotation.newPubKey
		data.VegaPubKeyNumber = rotation.newKeyNumber
		t.validators[nodeID] = data

		t.addProcessedKeyRotation(nodeID, oldPubKey, rotation.newPubKey, blockHeight)
		t.notifyKeyChange(ctx, oldPubKey, rotation.newPubKey)
		t.broker.Send(events.NewKeyRotationEvent(ctx, nodeID, oldPubKey, rotation.newPubKey, blockHeight))
	}

	delete(t.pendingPubKeyRotations, blockHeight)

	t.tss.changed = true
}
