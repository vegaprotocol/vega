// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

var ErrCurrentEthAddressDoesNotMatch = errors.New("current Ethereum address does not match")

type PendingEthereumKeyRotation struct {
	NodeID     string
	NewAddress string
}

type pendingEthereumKeyRotationMapping map[uint64][]PendingEthereumKeyRotation

func (pm pendingEthereumKeyRotationMapping) add(height uint64, rotation PendingEthereumKeyRotation) {
	if _, ok := pm[height]; !ok {
		pm[height] = []PendingEthereumKeyRotation{}
	}

	pm[height] = append(pm[height], rotation)
}

func (pm pendingEthereumKeyRotationMapping) get(height uint64) []PendingEthereumKeyRotation {
	rotations, ok := pm[height]
	if !ok {
		return []PendingEthereumKeyRotation{}
	}

	sort.Slice(rotations, func(i, j int) bool { return rotations[i].NodeID < rotations[j].NodeID })

	return rotations
}

func (t *Topology) RotateEthereumKey(
	ctx context.Context,
	nodeID string,
	currentBlockHeight uint64,
	kr *commandspb.EthereumKeyRotateSubmission,
) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.log.Debug("Adding Ethereum key rotation",
		logging.String("nodeID", nodeID),
		logging.Uint64("currentBlockHeight", currentBlockHeight),
		logging.Uint64("targetBlock", kr.TargetBlock),
		logging.String("newAddress", kr.NewAddress),
	)

	node, ok := t.validators[nodeID]
	if !ok {
		return fmt.Errorf("failed to rotate ethereum key for non existing validator %q", nodeID)
	}

	if currentBlockHeight >= kr.TargetBlock {
		return ErrTargetBlockHeightMustBeGraterThanCurrentHeight
	}

	if node.data.EthereumAddress != kr.CurrentAddress {
		return ErrCurrentEthAddressDoesNotMatch
	}

	toRemove := []NodeIDAddress{{NodeID: nodeID, EthAddress: node.data.EthereumAddress}}
	allValidators := t.validators.toNodeIDAdresses()

	// we can emit remove validator signatures immediately
	t.signatures.EmitRemoveValidatorsSignatures(ctx, toRemove, allValidators, t.timeService.GetTimeNow(), t.epochSeq)

	// schedule signature collection to future block
	// those signature should be emitted after validator has rotated is key in node wallet
	t.pendingEthKeyRotations.add(kr.TargetBlock, PendingEthereumKeyRotation{
		NodeID:     nodeID,
		NewAddress: kr.NewAddress,
	})

	t.log.Debug("Successfully added Ethereum key rotation to pending key rotations",
		logging.String("nodeID", nodeID),
		logging.Uint64("currentBlockHeight", currentBlockHeight),
		logging.Uint64("targetBlock", kr.TargetBlock),
		logging.String("newAddress", kr.NewAddress),
	)

	return nil
}

func (t *Topology) GetPendingEthereumKeyRotation(blockHeight uint64, nodeID string) *PendingEthereumKeyRotation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	rotations, ok := t.pendingEthKeyRotations[blockHeight]
	if !ok {
		return nil
	}

	for _, r := range rotations {
		if r.NodeID == nodeID {
			return &PendingEthereumKeyRotation{
				NodeID:     r.NodeID,
				NewAddress: r.NewAddress,
			}
		}
	}

	return nil
}

func (t *Topology) ethereumKeyRotationBeginBlockLocked(ctx context.Context) {
	t.log.Debug("Trying to apply pending Ethereum key rotations", logging.Uint64("currentBlockHeight", t.currentBlockHeight))

	// key swaps should run in deterministic order
	rotations := t.pendingEthKeyRotations.get(t.currentBlockHeight)
	if len(rotations) == 0 {
		return
	}

	t.log.Debug("Applying pending Ethereum key rotations", logging.Int("count", len(rotations)))

	for _, r := range rotations {
		t.log.Debug("Applying Ethereum key rotation",
			logging.String("nodeID", r.NodeID),
			logging.String("newAddress", r.NewAddress),
		)

		data, ok := t.validators[r.NodeID]
		if !ok {
			// this should actually happen if validator was removed due to poor performance
			t.log.Error("failed to rotate Ethereum key due to non present validator", logging.String("nodeID", r.NodeID), logging.String("EthereumAddress", r.NewAddress))
			continue
		}

		oldAddress := data.data.EthereumAddress

		data.data.EthereumAddress = r.NewAddress
		t.validators[r.NodeID] = data

		toAdd := []NodeIDAddress{{NodeID: r.NodeID, EthAddress: r.NewAddress}}
		t.signatures.EmitNewValidatorsSignatures(ctx, toAdd, t.timeService.GetTimeNow(), t.epochSeq)

		t.broker.Send(events.NewEthereumKeyRotationEvent(
			ctx,
			r.NodeID,
			oldAddress,
			r.NewAddress,
			t.currentBlockHeight,
		))

		t.log.Debug("Applied Ethereum key rotation",
			logging.String("nodeID", r.NodeID),
			logging.String("oldAddress", oldAddress),
			logging.String("newAddress", r.NewAddress),
		)
	}

	delete(t.pendingPubKeyRotations, t.currentBlockHeight)

	t.tss.changed = true
}
