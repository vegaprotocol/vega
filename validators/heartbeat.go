package validators

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sort"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"

	ecrypto "github.com/ethereum/go-ethereum/crypto"
)

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
	verifyEthSig func(message, signature []byte, hexAddress string) error) error {
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
	for _, v := range t.validators {
		// if the time since we've expected the heartbeat is too big,
		// we consider this validator invalid
		// arbitrary 500 seconds duration for the validator to send a
		// heartbeat, that's ~500 blocks a 1 block per sec
		hbExpired := len(v.heartbeatTracker.expectedNextHash) > 0 && v.heartbeatTracker.expectedNexthashSince.Add(500*time.Second).Before(t.currentTime)
		if hbExpired {
			v.heartbeatTracker.recordHeartbeatResult(false)
		}
	}
}

func (t *Topology) getNodesRequiringHB() []string {
	validatorNeedResend := []string{}
	for k, vs := range t.validators {
		if len(vs.heartbeatTracker.expectedNextHash) == 0 &&
			vs.heartbeatTracker.expectedNexthashSince.Add(1000*time.Second).Before(t.currentTime) &&
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
	validator.heartbeatTracker.expectedNexthashSince = t.currentTime

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

	ethereumSignature, err := t.wallets.GetEthereum().Sign(ecrypto.Keccak256(blockHashBytes))
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
		},
	}
}

// sendHeartbeat sends the hearbeat transaction.
func (t *Topology) sendHeartbeat(ctx context.Context, hb *commandspb.ValidatorHeartbeat) {
	t.cmd.CommandSync(ctx, txn.ValidatorHeartbeatCommand, hb, func(err error) {
		if err != nil {
			t.log.Error("couldn't send validator heartbeat", logging.Error(err))
			// we do a simple call again for it
			t.sendHeartbeat(ctx, hb)
		}
	})
}

// selectValidatorForHeartbeat selects a validator for sending heartbeat transaction.
func selectValidatorForHeartbeat(bhash string, validators []string) string {
	h := fnv.New64a()
	h.Write([]byte(bhash))
	index := h.Sum64() % uint64(len(validators))
	return validators[index]
}
