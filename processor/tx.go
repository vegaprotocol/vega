package processor

import (
	"encoding/hex"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/blockchain"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
)

const (
	TxPrefixLen  = 36
	TxCommandLen = 1
	TxHeaderLen  = TxPrefixLen + TxCommandLen
	TxValidLen   = TxHeaderLen
)

var (
	ErrInvalidTxPayloadLen = errors.New("payload size is incorrec, should be > 37 bytes")
)

type Tx struct {
	tx  *types.Transaction
	sig *types.Signature
}

func NewTx(tx *types.Transaction, sig *types.Signature) (*Tx, error) {
	if len(tx.InputData) < TxValidLen {
		return nil, ErrInvalidTxPayloadLen
	}

	return &Tx{tx, sig}, nil
}

// Hash returns hash of the given Tx. Hashes are unique to every vega tx.
// The hash is the first TxHeaderLen bytes.
func (tx *Tx) Hash() []byte { return tx.tx.InputData[:TxHeaderLen] }

// Payload returns the payload of the transaction, this is all the bytes,
// excluding the prefix and the command.
func (tx *Tx) Payload() []byte { return tx.tx.InputData[TxHeaderLen:] }

func (tx *Tx) PubKey() []byte { return tx.tx.GetPubKey() }

// Command returns the Command of the Tx
func (tx *Tx) Command() blockchain.Command {
	cmd := tx.tx.InputData[TxPrefixLen]
	return blockchain.Command(cmd)
}

func (tx *Tx) Unmarshal(i interface{}) error {
	if t, ok := i.(proto.Message); ok {
		return proto.Unmarshal(tx.Payload(), t)
	}
	return nil
}

// toProto decodes a tx given its command into the respective proto type
func (tx *Tx) toProto() (interface{}, error) {
	msgs := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand:   &types.OrderSubmission{},
		blockchain.CancelOrderCommand:   &types.OrderCancellation{},
		blockchain.AmendOrderCommand:    &types.OrderAmendment{},
		blockchain.ProposeCommand:       &types.Proposal{},
		blockchain.VoteCommand:          &types.Vote{},
		blockchain.NodeVoteCommand:      &types.NodeVote{},
		blockchain.WithdrawCommand:      &types.Withdraw{},
		blockchain.RegisterNodeCommand:  &types.NodeRegistration{},
		blockchain.NodeSignatureCommand: &types.NodeSignature{},
		blockchain.ChainEventCommand:    &types.ChainEvent{},
	}
	msg, ok := msgs[tx.Command()]
	if !ok {
		return nil, fmt.Errorf("don't know how to unmarshal command '%s'", tx.Command().String())
	}

	if err := tx.Unmarshal(msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// Validate verifies that the pubkey matches
func (tx *Tx) Validate() error {
	cmd, err := tx.toProto()
	if err != nil {
		return err
	}

	pubkey := hex.EncodeToString(tx.PubKey())
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

func (tx *Tx) asOrderSubmission() (*types.Order, error) {
	submission := &types.OrderSubmission{}
	err := proto.Unmarshal(tx.Payload(), submission)
	if err != nil {
		return nil, err
	}

	order := types.Order{
		Id:          submission.Id,
		MarketID:    submission.MarketID,
		PartyID:     submission.PartyID,
		Price:       submission.Price,
		Size:        submission.Size,
		Side:        submission.Side,
		TimeInForce: submission.TimeInForce,
		Type:        submission.Type,
		ExpiresAt:   submission.ExpiresAt,
		Reference:   submission.Reference,
		Status:      types.Order_STATUS_ACTIVE,
		CreatedAt:   0,
		Remaining:   submission.Size,
	}

	return &order, nil
}
