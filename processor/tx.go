package processor

import (
	"encoding/hex"
	"errors"

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
	proto *types.Transaction
}

func NewTx(tx *types.Transaction) (*Tx, error) {
	if len(tx.InputData) < TxValidLen {
		return nil, ErrInvalidTxPayloadLen
	}

	return &Tx{tx}, nil
}

// Payload returns the payload of the transaction, this is all the bytes,
// excluding the prefix and the command.
func (tx *Tx) Payload() []byte { return tx.proto.InputData[TxHeaderLen:] }

func (tx *Tx) PubKey() []byte { return tx.proto.GetPubKey() }

// Command returns the Command of the Tx
func (tx *Tx) Command() blockchain.Command {
	cmd := tx.proto.InputData[TxPrefixLen]
	return blockchain.Command(cmd)
}

// Validate returns error if the Tx is invalid.
func (tx *Tx) Validate() error {
	switch tx.Command() {
	case blockchain.SubmitOrderCommand:
		order, err := tx.asOrderSubmission()
		if err != nil {
			return err
		}

		if order.PartyID != hex.EncodeToString(tx.PubKey()) {
			return ErrOrderSubmissionPartyAndPubKeyDoesNotMatch
		}

		return nil

	case blockchain.CancelOrderCommand:
		order, err := tx.asOrderCancellation()
		if err != nil {
			return err
		}
		_ = order
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

func (tx *Tx) asOrderCancellation() (*types.OrderCancellation, error) {
	order := &types.OrderCancellation{}
	err := proto.Unmarshal(tx.Payload(), order)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (tx *Tx) asWithdraw() (*types.Withdraw, error) {
	w := &types.Withdraw{}
	err := proto.Unmarshal(tx.Payload(), w)
	if err != nil {
		return nil, err
	}
	return w, nil
}
