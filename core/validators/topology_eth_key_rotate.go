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
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/nodewallets/eth/clef"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrCurrentEthAddressDoesNotMatch = errors.New("current Ethereum address does not match")
	ErrCannotRotateToSameKey         = errors.New("new Ethereum address cannot be the same as the previous Ethereum address")
	ErrNodeHasUnresolvedRotation     = errors.New("ethereum keys from a previous rotation have not been resolved on the multisig control contract")
)

type PendingEthereumKeyRotation struct {
	NodeID           string
	NewAddress       string
	OldAddress       string
	SubmitterAddress string
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

func (t *Topology) hasPendingEthKeyRotation(nodeID string) bool {
	for _, rotations := range t.pendingEthKeyRotations {
		for _, r := range rotations {
			if r.NodeID == nodeID {
				return true
			}
		}
	}
	return false
}

func (t *Topology) ProcessEthereumKeyRotation(
	ctx context.Context,
	publicKey string,
	kr *commandspb.EthereumKeyRotateSubmission,
	verify func(message, signature []byte, hexAddress string) error,
) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.log.Debug("Received ethereum key rotation",
		logging.String("vega-pub-key", publicKey),
		logging.String("newAddress", kr.NewAddress),
		logging.Uint64("currentBlockHeight", t.currentBlockHeight),
		logging.Uint64("targetBlock", kr.TargetBlock),
	)

	var node *valState
	for _, v := range t.validators {
		if v.data.VegaPubKey == publicKey {
			node = v
			break
		}
	}

	if node == nil {
		err := fmt.Errorf("failed to rotate ethereum key for non existing validator %q", publicKey)
		t.log.Debug("Failed to add Eth key rotation", logging.Error(err))
		return err
	}

	if err := t.validateRotation(kr, node.data, verify); err != nil {
		return err
	}

	// schedule the key rotation to a future block
	t.pendingEthKeyRotations.add(kr.TargetBlock,
		PendingEthereumKeyRotation{
			NodeID:           node.data.ID,
			NewAddress:       kr.NewAddress,
			OldAddress:       kr.CurrentAddress,
			SubmitterAddress: kr.SubmitterAddress,
		})

	t.log.Debug("Successfully added Ethereum key rotation to pending key rotations",
		logging.String("vega-pub-key", publicKey),
		logging.Uint64("currentBlockHeight", t.currentBlockHeight),
		logging.Uint64("targetBlock", kr.TargetBlock),
		logging.String("newAddress", kr.NewAddress),
	)

	if node.status != ValidatorStatusTendermint {
		return nil
	}

	toRemove := []NodeIDAddress{{NodeID: node.data.ID, EthAddress: node.data.EthereumAddress}}
	t.signatures.PrepareValidatorSignatures(ctx, toRemove, t.epochSeq, false)
	if len(kr.SubmitterAddress) != 0 {
		// we were given an address that will be submitting the multisig changes, we can emit a remove signature for it right now
		t.signatures.EmitValidatorRemovedSignatures(ctx, kr.SubmitterAddress, node.data.ID, t.timeService.GetTimeNow())
	}

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
	// check any unfinalised key rotations
	remove := []string{}
	for _, r := range t.unresolvedEthKeyRotations {
		if !t.multiSigTopology.IsSigner(r.OldAddress) && t.multiSigTopology.IsSigner(r.NewAddress) {
			t.log.Info("ethereum key rotations have been resolved on the multisig contract", logging.String("nodeID", r.NodeID), logging.String("old-address", r.OldAddress))
			remove = append(remove, r.NodeID)
		}
	}

	for _, nodeID := range remove {
		delete(t.unresolvedEthKeyRotations, nodeID)
	}

	// key swaps should run in deterministic order
	rotations := t.pendingEthKeyRotations.get(t.currentBlockHeight)
	if len(rotations) == 0 {
		return
	}

	t.log.Debug("Applying ethereum key-rotations", logging.Uint64("currentBlockHeight", t.currentBlockHeight), logging.Int("n-rotations", len(rotations)))
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

		if data.status != ValidatorStatusTendermint {
			continue
		}

		toAdd := []NodeIDAddress{{NodeID: r.NodeID, EthAddress: r.NewAddress}}
		t.signatures.PrepareValidatorSignatures(ctx, toAdd, t.epochSeq, true)

		if len(r.SubmitterAddress) != 0 {
			// we were given an address that will be submitting the multisig changes, we can emit signatures for it right now
			t.signatures.EmitValidatorAddedSignatures(ctx, r.SubmitterAddress, r.NodeID, t.timeService.GetTimeNow())
		}

		// add to unfinalised map so we can wait to see the changes on the contract
		t.unresolvedEthKeyRotations[data.data.ID] = r
	}

	delete(t.pendingEthKeyRotations, t.currentBlockHeight)
}

func (t *Topology) validateRotation(kr *commandspb.EthereumKeyRotateSubmission, data ValidatorData, verify func(message, signature []byte, hexAddress string) error) error {
	if t.hasPendingEthKeyRotation(data.ID) {
		return ErrNodeAlreadyHasPendingKeyRotation
	}

	if _, ok := t.unresolvedEthKeyRotations[data.ID]; ok {
		return ErrNodeHasUnresolvedRotation
	}

	if t.currentBlockHeight >= kr.TargetBlock {
		t.log.Debug("target block height is not above current block height", logging.Uint64("target", kr.TargetBlock), logging.Uint64("current", t.currentBlockHeight))
		return ErrTargetBlockHeightMustBeGreaterThanCurrentHeight
	}

	if data.EthereumAddress != kr.CurrentAddress {
		t.log.Debug("current addresses do not match", logging.String("current", data.EthereumAddress), logging.String("submitted", kr.CurrentAddress))
		return ErrCurrentEthAddressDoesNotMatch
	}

	if data.EthereumAddress == kr.NewAddress {
		t.log.Debug("trying to rotate to the same key", logging.String("current", data.EthereumAddress), logging.String("new-address", kr.NewAddress))
		return ErrCannotRotateToSameKey
	}

	if err := VerifyEthereumKeyRotation(kr, verify); err != nil {
		return err
	}

	return nil
}

func SignEthereumKeyRotation(
	kr *commandspb.EthereumKeyRotateSubmission,
	ethSigner Signer,
) error {
	buf, err := makeEthKeyRotationSignableMessage(kr)
	if err != nil {
		return err
	}

	if ethSigner.Algo() != clef.ClefAlgoType {
		buf = crypto.Keccak256(buf)
	}
	ethereumSignature, err := ethSigner.Sign(buf)
	if err != nil {
		return err
	}

	kr.EthereumSignature = &commandspb.Signature{
		Value: hex.EncodeToString(ethereumSignature),
		Algo:  ethSigner.Algo(),
	}

	return nil
}

func VerifyEthereumKeyRotation(kr *commandspb.EthereumKeyRotateSubmission, verify func(message, signature []byte, hexAddress string) error) error {
	buf, err := makeEthKeyRotationSignableMessage(kr)
	if err != nil {
		return err
	}

	eths, err := hex.DecodeString(kr.GetEthereumSignature().Value)
	if err != nil {
		return err
	}

	if err := verify(buf, eths, kr.NewAddress); err != nil {
		return err
	}

	return nil
}

func makeEthKeyRotationSignableMessage(kr *commandspb.EthereumKeyRotateSubmission) ([]byte, error) {
	if len(kr.CurrentAddress) <= 0 || len(kr.NewAddress) <= 0 || kr.TargetBlock == 0 {
		return nil, ErrMissingRequiredAnnounceNodeFields
	}

	msg := kr.CurrentAddress + kr.NewAddress + fmt.Sprintf("%d", kr.TargetBlock)

	return []byte(msg), nil
}
