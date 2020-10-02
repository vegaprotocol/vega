package processor

import (
	"encoding/hex"
	"errors"
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/tx"

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
	tx *types.Transaction
}

func NewTx(tx *types.Transaction) (*Tx, error) {
	if len(tx.InputData) < TxHeaderLen {
		return nil, ErrInvalidTxPayloadLen
	}

	return &Tx{tx}, nil
}

// Hash returns hash of the given Tx. Hashes are unique to every vega tx.
// The hash is the first TxHeaderLen bytes.
func (tx *Tx) Hash() []byte { return tx.tx.InputData[:TxHashLen] }

func (tx *Tx) PubKey() []byte { return tx.tx.GetPubKey() }

// Command returns the Command of the Tx
func (t *Tx) Command() tx.Command {
	cmd := t.tx.InputData[TxHashLen]
	return tx.Command(cmd)
}

// payload returns the payload of the transaction, this is all the bytes,
// excluding the prefix and the command.
func (tx *Tx) payload() []byte { return tx.tx.InputData[TxHeaderLen:] }

func (tx *Tx) Unmarshal(i interface{}) error {
	if t, ok := i.(proto.Message); ok {
		return proto.Unmarshal(tx.payload(), t)
	}
	return nil
}

// toProto decodes a tx given its command into the respective proto type
func (t *Tx) toProto() (interface{}, error) {
	msgs := map[tx.Command]proto.Message{
		tx.SubmitOrderCommand:   &types.OrderSubmission{},
		tx.CancelOrderCommand:   &types.OrderCancellation{},
		tx.AmendOrderCommand:    &types.OrderAmendment{},
		tx.ProposeCommand:       &types.Proposal{},
		tx.VoteCommand:          &types.Vote{},
		tx.NodeVoteCommand:      &types.NodeVote{},
		tx.WithdrawCommand:      &types.WithdrawSubmission{},
		tx.RegisterNodeCommand:  &types.NodeRegistration{},
		tx.NodeSignatureCommand: &types.NodeSignature{},
		tx.ChainEventCommand:    &types.ChainEvent{},
	}
	msg, ok := msgs[t.Command()]
	if !ok {
		return nil, fmt.Errorf("don't know how to unmarshal command '%s'", t.Command().String())
	}

	if err := t.Unmarshal(msg); err != nil {
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
	err := proto.Unmarshal(tx.payload(), submission)
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
