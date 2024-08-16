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
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/nodewallets/eth/clef"
	"code.vegaprotocol.io/vega/core/txn"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/cenkalti/backoff"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
)

var ErrHeartbeatHasExpired = errors.New("heartbeat received after expiry")

// validatorHeartbeatTracker keeps track of heartbeat transactions and their results.
type validatorHeartbeatTracker struct {
	// the next hash expected for this validator to sign
	expectedNextHash string
	// the time at which we've seen the hash
	expectedNexthashSince time.Time
	// the index to the last 10 signatures
	blockIndex int
	// last 10 signatures
	blockSigs [10]bool
}

// recordHeartbeatResult records the result of an expected signature
// if true it means that the validator has signed the correct block within a reasonable time
// otherwise they either didn't sign on time or didn't sign properly.
func (v *validatorHeartbeatTracker) recordHeartbeatResult(status bool) {
	v.blockSigs[v.blockIndex%10] = status
	v.blockIndex++
	v.expectedNextHash = ""
}

// ProcessValidatorHeartbeat is verifying the signatures from a validator's transaction and records the status.
func (t *Topology) ProcessValidatorHeartbeat(ctx context.Context, vh *commandspb.ValidatorHeartbeat,
	verifyVegaSig func(message, signature, pubkey []byte) error,
	verifyEthSig func(message, signature []byte, hexAddress string) error,
) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	validator, ok := t.validators[vh.NodeId]
	if !ok {
		return fmt.Errorf("received an heartbeat from a non-validator node: %v", vh.NodeId)
	}

	var (
		node = t.validators[vh.NodeId]
		hash = validator.heartbeatTracker.expectedNextHash
	)

	if hash != vh.Message {
		// the heartbeat came in too late, we're already waiting for another one
		return ErrHeartbeatHasExpired
	}

	vegas, err := hex.DecodeString(vh.GetVegaSignature().Value)
	if err != nil {
		validator.heartbeatTracker.recordHeartbeatResult(false)
		return err
	}
	vegaPubKey, err := hex.DecodeString(node.data.VegaPubKey)
	if err != nil {
		validator.heartbeatTracker.recordHeartbeatResult(false)
		return err
	}
	if err := verifyVegaSig([]byte(hash), vegas, vegaPubKey); err != nil {
		validator.heartbeatTracker.recordHeartbeatResult(false)
		return err
	}

	eths, err := hex.DecodeString(vh.GetEthereumSignature().Value)
	if err != nil {
		validator.heartbeatTracker.recordHeartbeatResult(false)
		return err
	}

	if err := verifyEthSig([]byte(hash), eths, node.data.EthereumAddress); err != nil {
		validator.heartbeatTracker.recordHeartbeatResult(false)
		return err
	}

	// record the success
	validator.heartbeatTracker.recordHeartbeatResult(true)

	return nil
}

// checkHeartbeat checks if there's a validator who is late on their heartbeat transaction and checks if any validator needs to send a heartbeat transaction.
// if so and this validator is *this* then it sends the transaction.
func (t *Topology) checkHeartbeat(ctx context.Context) {
	// this is called TraceID but is actually the block hash...
	_, bhash := vgcontext.TraceIDFromContext(ctx)
	t.checkHeartbeatWithBlockHash(ctx, bhash)
}

// checkAndExpireStaleHeartbeats checks if there is a validator with stale heartbeat and records the failure.
func (t *Topology) checkAndExpireStaleHeartbeats() {
	// if a node hasn't sent a heartbeat when they were expected, record the failure and reset their state.
	now := t.timeService.GetTimeNow()
	for _, v := range t.validators {
		// if the time since we've expected the heartbeat is too big,
		// we consider this validator invalid
		// arbitrary 500 seconds duration for the validator to send a
		// heartbeat, that's ~500 blocks a 1 block per sec
		hbExpired := len(v.heartbeatTracker.expectedNextHash) > 0 && v.heartbeatTracker.expectedNexthashSince.Add(t.timeToSendHeartbeat).Before(now)
		if hbExpired {
			v.heartbeatTracker.recordHeartbeatResult(false)
		}
	}
}

func (t *Topology) getNodesRequiringHB() []string {
	validatorNeedResend := []string{}
	now := t.timeService.GetTimeNow()
	for k, vs := range t.validators {
		if len(vs.heartbeatTracker.expectedNextHash) == 0 &&
			vs.heartbeatTracker.expectedNexthashSince.Add(t.timeBetweenHeartbeats).Before(now) &&
			vs.data.FromEpoch <= t.epochSeq {
			validatorNeedResend = append(validatorNeedResend, k)
		}
	}
	sort.Strings(validatorNeedResend)
	return validatorNeedResend
}

func (t *Topology) checkHeartbeatWithBlockHash(ctx context.Context, bhash string) {
	t.checkAndExpireStaleHeartbeats()

	// check which node
	validatorNeedResend := t.getNodesRequiringHB()
	if len(validatorNeedResend) == 0 {
		return
	}

	// select deterministically which validator would send a heartbeat this round if they need to.
	selectedValidator := selectValidatorForHeartbeat(bhash, validatorNeedResend)
	validator := t.validators[selectedValidator]

	// time for another round
	validator.heartbeatTracker.expectedNextHash = bhash
	validator.heartbeatTracker.expectedNexthashSince = t.timeService.GetTimeNow()

	// now we figure out if we need to send a heartbeat now
	if !t.isValidatorSetup || selectedValidator != t.SelfNodeID() {
		// not a validator, go home
		return
	}

	if hb := t.prepareHeartbeat(bhash); hb != nil {
		t.sendHeartbeat(ctx, hb)
	}
}

// prepareHeartbeat prepares a heartbeat transaction.
func (t *Topology) prepareHeartbeat(blockHash string) *commandspb.ValidatorHeartbeat {
	blockHashBytes := []byte(blockHash)
	vegaSignature, err := t.wallets.GetVega().Sign(blockHashBytes)
	if err != nil {
		t.log.Error("could not sign heartbeat with vega wallet",
			logging.String("block-hash", blockHash),
			logging.Error(err),
		)
		return nil
	}

	signer := t.wallets.GetEthereum()
	if signer.Algo() != clef.ClefAlgoType {
		// hash our message before signing it
		blockHashBytes = ecrypto.Keccak256(blockHashBytes)
	}
	ethereumSignature, err := signer.Sign(blockHashBytes)
	if err != nil {
		t.log.Error("could not sign heartbeat with ethereum wallet",
			logging.String("block-hash", blockHash),
			logging.Error(err),
		)
		return nil
	}

	return &commandspb.ValidatorHeartbeat{
		NodeId: t.SelfNodeID(),
		VegaSignature: &commandspb.Signature{
			Value: hex.EncodeToString(vegaSignature),
			Algo:  t.wallets.GetVega().Algo(),
		},
		EthereumSignature: &commandspb.Signature{
			Value: hex.EncodeToString(ethereumSignature),
			Algo:  signer.Algo(),
		},
		Message: blockHash,
	}
}

// sendHeartbeat sends the hearbeat transaction.
func (t *Topology) sendHeartbeat(ctx context.Context, hb *commandspb.ValidatorHeartbeat) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = t.timeToSendHeartbeat
	bo.InitialInterval = 1 * time.Second

	t.log.Debug("sending heartbeat", logging.String("nodeID", hb.NodeId))
	t.cmd.CommandSync(ctx, txn.ValidatorHeartbeatCommand, hb, func(_ string, err error) {
		if err != nil {
			//			t.log.Error("couldn't send validator heartbeat", logging.Error(err))
			return
		}
		t.log.Debug("heartbeat sent", logging.String("nodeID", hb.NodeId))
	}, bo)
}

// selectValidatorForHeartbeat selects a validator for sending heartbeat transaction.
func selectValidatorForHeartbeat(bhash string, validators []string) string {
	h := fnv.New64a()
	h.Write([]byte(bhash))
	index := h.Sum64() % uint64(len(validators))
	return validators[index]
}
