package processor

import (
	"encoding/hex"
	"errors"
	"fmt"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
)

const (
	TxHashLen    = 36
	TxCommandLen = 1
	TxHeaderLen  = TxHashLen + TxCommandLen
)

var (
	ErrInvalidTxPayloadLen = errors.New("payload size is incorrect, should be > 37 bytes")
)

type Tx struct {
	tx        *types.Transaction
	signature []byte
}

func NewTx(tx *types.Transaction, signature []byte) (*Tx, error) {
	if len(tx.InputData) < TxHeaderLen {
		return nil, ErrInvalidTxPayloadLen
	}

	return &Tx{tx, signature}, nil
}

// Hash returns hash of the given Tx. Hashes are unique to every vega tx.
// The hash is the first TxHeaderLen bytes.
func (t *Tx) Hash() []byte { return t.tx.InputData[:TxHashLen] }

// PubKey returns the Tx's public key.
func (t *Tx) PubKey() []byte { return t.tx.GetPubKey() }

// BlockHeight returns the target block for which the Tx has been broadcasted.
// The Tx might be included on a higher block height.
// Depending on the tolerance of the chain the Tx might be included or rejected.
func (t *Tx) BlockHeight() uint64 { return t.tx.BlockHeight }

func (t *Tx) Signature() []byte { return t.signature }

// Command returns the Command of the Tx
func (t *Tx) Command() txn.Command {
	cmd := t.tx.InputData[TxHashLen]
	return txn.Command(cmd)
}

// payload returns the payload of the transaction, this is all the bytes,
// excluding the prefix and the command.
func (t *Tx) payload() []byte { return t.tx.InputData[TxHeaderLen:] }

func (t *Tx) Unmarshal(i interface{}) error {
	if msg, ok := i.(proto.Message); ok {
		return proto.Unmarshal(t.payload(), msg)
	}
	return nil
}

// toProto decodes a tx given its command into the respective proto type
func (t *Tx) toProto() (interface{}, error) {
	var msg proto.Message
	switch t.Command() {
	case txn.SubmitOrderCommand:
		msg = &types.OrderSubmission{}
	case txn.CancelOrderCommand:
		msg = &types.OrderCancellation{}
	case txn.AmendOrderCommand:
		msg = &types.OrderAmendment{}
	case txn.ProposeCommand:
		msg = &types.Proposal{}
	case txn.VoteCommand:
		msg = &types.Vote{}
	case txn.NodeVoteCommand:
		msg = &types.NodeVote{}
	case txn.WithdrawCommand:
		msg = &types.WithdrawSubmission{}
	case txn.RegisterNodeCommand:
		msg = &types.NodeRegistration{}
	case txn.NodeSignatureCommand:
		msg = &types.NodeSignature{}
	case txn.LiquidityProvisionCommand:
		msg = &types.LiquidityProvisionSubmission{}
	case txn.ChainEventCommand:
		msg = &types.ChainEvent{}
	default:
		return nil, fmt.Errorf("don't know how to unmarshal command '%s'", t.Command().String())
	}

	if err := t.Unmarshal(msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// Validate verifies that the pubkey matches
func (t *Tx) Validate() error {
	cmd, err := t.toProto()
	if err != nil {
		return err
	}

	pubkey := hex.EncodeToString(t.PubKey())
	// Verify party ID on those types who have it.
	if t, ok := cmd.(interface{ GetPartyID() string }); ok {
		if t.GetPartyID() != pubkey {
			return errors.New("pubkey does not match with party-id")
		}
	}

	switch t := cmd.(type) {
	case *types.NodeRegistration:
		if hex.EncodeToString(t.PubKey) != pubkey {
			return errors.New("pubkey mismatch")
		}
	}

	return nil
}
