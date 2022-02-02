package validators

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"

	ecrypto "github.com/ethereum/go-ethereum/crypto"
)

type validatorHeartbeatTracker struct {
	// the last time a validator signed a heartbeat
	lastHeartbeat time.Time

	// the next hash expected for this validator to sign
	expectedNextHash string
	// the time at which we've seen the hash
	expectedNexthashSince time.Time
}

func (v *validatorHeartbeatTracker) IsValid(currentTime time.Time) bool {
	// until we get the first heartbeat we assume so.
	if v.lastHeartbeat.IsZero() {
		return true
	}

	// if the time since we've expected the heartbeat is too big,
	// we consider this validator invalid
	// arbitrary 500 secondss duration for the validator to send a
	// heartbeat, that's ~500 blocks a 1 block per sec
	return v.expectedNexthashSince.Add(500 * time.Second).Before(currentTime)
}

func (t *Topology) ProcessValidatorHeartbeat(
	ctx context.Context, vh *commandspb.ValidatorHeartbeat) error {
	heartBeat, ok := t.heartBeats[vh.NodeId]
	if !ok {
		return fmt.Errorf("received an heartbeat from a non-validator node: %v", vh.NodeId)
	}

	var (
		// now we get the node
		// TODO(Zohar): needs to eventually get from both a list of
		// validators and erzats, now just doing from the node list
		node = t.validators[vh.NodeId]
		hash = heartBeat.expectedNextHash
	)

	vegas, err := hex.DecodeString(vh.GetVegaSignature().Value)
	if err != nil {
		return err
	}
	vegaPubKey, err := hex.DecodeString(node.VegaPubKey)
	if err != nil {
		return err
	}
	if err := vgcrypto.VerifyVegaSignature([]byte(hash), vegas, vegaPubKey); err != nil {
		return err
	}

	eths, err := hex.DecodeString(vh.GetEthereumSignature().Value)
	if err != nil {
		return err
	}

	if err := vgcrypto.VerifyEthereumSignature([]byte(hash), eths, node.EthereumAddress); err != nil {
		return err
	}

	// if we reach this point, the signature were valid,
	// we can reset the heartbeat struct
	heartBeat.expectedNextHash = ""
	heartBeat.expectedNexthashSince = time.Time{}

	return nil
}

func (t *Topology) checkHeartbeat(ctx context.Context) {
	// this is called TraceID but is actually the block hash...
	_, bhash := vgcontext.TraceIDFromContext(ctx)

	// now figure out if a node hasn't send the heartbeats
	for _, v := range t.heartBeats {
		if !v.IsValid(t.currentTime) {
			// TODO(Zohar): do stuff here, but the code doesn't exist in this pr
			continue
		}
	}

	selectedValidator := t.selectValidatorForHeartbeat(bhash)

	// now we get the validator, and check if it's been more than
	// 1000 block since its last hearbeat
	heartbeatTracker, ok := t.heartBeats[selectedValidator]
	if !ok {
		t.log.Panic("node heartbeatTracker is not initialied", logging.String("node-id", selectedValidator))
	}

	// if the hearbeats happened ~+1000 secs ago
	// and there's no expected hash, we can update them
	if heartbeatTracker.lastHeartbeat.Add(1000*time.Second).Before(t.currentTime) &&
		len(heartbeatTracker.expectedNextHash) <= 0 {
		heartbeatTracker.expectedNextHash = bhash
		heartbeatTracker.expectedNexthashSince = t.currentTime
	}

	// now we figure out if we need to send a heartbeat now
	if !t.isValidatorSetup || selectedValidator != t.SelfNodeID() {
		// not a validator, or expecting to become one, we are done
		return
	}

	if hb := t.prepareHeartbeat(bhash); hb != nil {
		t.sendHeartbeat(ctx, hb)
	}
}

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

func (t *Topology) sendHeartbeat(
	ctx context.Context, hb *commandspb.ValidatorHeartbeat) {
	t.cmd.CommandSync(ctx, txn.ValidatorHeartbeatCommand, hb, func(err error) {
		t.log.Error("couldn't send validator heartbeat", logging.Error(err))
		// we do a simple call again for it
		t.sendHeartbeat(ctx, hb)
	})
}

func (t *Topology) selectValidatorForHeartbeat(bhash string) string {
	h := fnv.New64a()
	h.Write([]byte(bhash))
	validators := t.validators.GetSortedListedINodeIDs()
	index := h.Sum64() % uint64(len(validators))
	return validators[index]
}
