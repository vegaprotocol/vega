package processor_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/processor"
	types "code.vegaprotocol.io/vega/proto"

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

type TxTestSuite struct {
}

func (s *TxTestSuite) testValidateCommandSuccess(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	msgs := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		blockchain.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		blockchain.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		blockchain.VoteCommand: &types.Vote{
			PartyID: party,
		},
		blockchain.WithdrawCommand: &types.Withdraw{
			PartyID: party,
		},
		blockchain.ProposeCommand: &types.Proposal{
			PartyID: party,
		},
	}

	for cmd, msg := range msgs {
		// Build the Tx
		var hash [processor.TxHashLen]byte // empty hash works for this
		payload, err := proto.Marshal(msg)
		require.NoError(t, err)

		input := concatBytes(
			hash[:],
			[]byte{byte(cmd)},
			payload,
		)

		rawTx := &types.Transaction{
			InputData: input,
			From: &types.Transaction_PubKey{
				PubKey: key,
			},
		}
		tx, err := processor.NewTx(rawTx)
		require.NoError(t, err)

		require.NoError(t, tx.Validate())
	}
}

func (s *TxTestSuite) testValidateCommandsFail(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString([]byte("another-party"))
	msgs := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		blockchain.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		blockchain.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		blockchain.VoteCommand: &types.Vote{
			PartyID: party,
		},
		blockchain.WithdrawCommand: &types.Withdraw{
			PartyID: party,
		},
		blockchain.ProposeCommand: &types.Proposal{
			PartyID: party,
		},
	}

	for cmd, msg := range msgs {
		// Build the Tx
		var hash [processor.TxHashLen]byte // empty hash works for this
		payload, err := proto.Marshal(msg)
		require.NoError(t, err)

		input := concatBytes(
			hash[:],
			[]byte{byte(cmd)},
			payload,
		)

		rawTx := &types.Transaction{
			InputData: input,
			From: &types.Transaction_PubKey{
				PubKey: key,
			},
		}
		tx, err := processor.NewTx(rawTx)
		require.NoError(t, err)

		require.Error(t, tx.Validate())
	}
}

func (s *TxTestSuite) testValidateSignedInvalidCommand(t *testing.T) {
	cmd := blockchain.VoteCommand
	party := []byte("party-id")
	// wrong type for this command
	payload, err := proto.Marshal(&types.Proposal{
		ID:        "XXX",
		PartyID:   hex.EncodeToString(party),
		Reference: "some-reference",
	})

	var hash [processor.TxHashLen]byte // empty hash works for this
	input := concatBytes(
		hash[:],
		[]byte{byte(cmd)},
		payload,
	)
	rawTx := &types.Transaction{
		InputData: input,
	}
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
					[]byte{byte(blockchain.SubmitOrderCommand)},
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
