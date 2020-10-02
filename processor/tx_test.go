package processor_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/processor"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/tx"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func concatBytes(bzs ...[]byte) []byte {
	buf := bytes.NewBuffer(nil)
	for _, bz := range bzs {
		_, err := buf.Write(bz)
		if err != nil {
			panic(err)
		}
	}
	return buf.Bytes()
}

func txEncode(t *testing.T, cmd tx.Command, msg proto.Message) *types.Transaction {
	var hash [processor.TxHashLen]byte // empty hash works for this
	payload, err := proto.Marshal(msg)
	require.NoError(t, err)

	bz := concatBytes(
		hash[:],
		[]byte{byte(cmd)},
		payload,
	)

	return &types.Transaction{
		InputData: bz,
	}
}

type TxTestSuite struct {
}

func (s *TxTestSuite) testValidateCommandSuccess(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	msgs := map[tx.Command]proto.Message{
		tx.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		tx.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		tx.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		tx.VoteCommand: &types.Vote{
			PartyID: party,
		},
		tx.WithdrawCommand: &types.WithdrawSubmission{
			PartyID: party,
		},
		tx.ProposeCommand: &types.Proposal{
			PartyID: party,
		},
	}

	for cmd, msg := range msgs {
		// Build the Tx
		rawTx := txEncode(t, cmd, msg)
		rawTx.From = &types.Transaction_PubKey{
			PubKey: key,
		}
		tx, err := processor.NewTx(rawTx)
		require.NoError(t, err)

		require.NoError(t, tx.Validate())
	}
}

func (s *TxTestSuite) testValidateCommandsFail(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString([]byte("another-party"))
	msgs := map[tx.Command]proto.Message{
		tx.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		tx.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		tx.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		tx.VoteCommand: &types.Vote{
			PartyID: party,
		},
		tx.WithdrawCommand: &types.WithdrawSubmission{
			PartyID: party,
		},
		tx.ProposeCommand: &types.Proposal{
			PartyID: party,
		},
	}

	for cmd, msg := range msgs {
		// Build the Tx
		rawTx := txEncode(t, cmd, msg)
		rawTx.From = &types.Transaction_PubKey{
			PubKey: key,
		}
		tx, err := processor.NewTx(rawTx)
		require.NoError(t, err)

		require.Error(t, tx.Validate())
	}
}

func (s *TxTestSuite) testValidateSignedInvalidCommand(t *testing.T) {
	cmd := tx.VoteCommand
	party := []byte("party-id")
	// wrong type for this command
	prop := &types.Proposal{
		ID:        "XXX",
		PartyID:   hex.EncodeToString(party),
		Reference: "some-reference",
	}

	rawTx := txEncode(t, cmd, prop)
	tx, err := processor.NewTx(rawTx)
	require.NoError(t, err)

	assert.Error(t, tx.Validate())
}

func (s *TxTestSuite) testValidateSignedInvalidPayload(t *testing.T) {
	t.Run("TooShort", func(t *testing.T) {
		_, err := processor.NewTx(
			&types.Transaction{
				InputData: []byte("shorter-than-37-bytes"),
			},
		)
		require.Error(t, err)
	})

	t.Run("RandomCrap", func(t *testing.T) {
		var hash [processor.TxHashLen]byte
		tx, err := processor.NewTx(
			&types.Transaction{
				InputData: concatBytes(
					hash[:],
					[]byte{byte(tx.SubmitOrderCommand)},
					[]byte("foobar"),
				),
			},
		)
		require.NoError(t, err)
		require.Error(t, tx.Validate())
	})
}

func TestTxValidation(t *testing.T) {
	s := &TxTestSuite{}

	t.Run("Test all signed commands basic - success", s.testValidateCommandSuccess)
	t.Run("Test all signed commands basic - failure", s.testValidateCommandsFail)
	t.Run("Test validate signed invalid command", s.testValidateSignedInvalidCommand)
	t.Run("Test validate signed invalid payload", s.testValidateSignedInvalidPayload)
}
