package processor_test

import (
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/processor/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

type procTest struct {
	*processor.Processor
	eng    *mocks.MockExecutionEngine
	ts     *mocks.MockTimeService
	tickCB func(time.Time)
	ctrl   *gomock.Controller
}

func getTestProcessor(t *testing.T) *procTest {
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	eng := mocks.NewMockExecutionEngine(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	var cb func(time.Time)
	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1).Do(func(c func(time.Time)) {
		cb = c
	})
	proc := processor.New(log, processor.NewDefaultConfig(), eng, ts)
	return &procTest{
		Processor: proc,
		eng:       eng,
		ts:        ts,
		tickCB:    cb,
		ctrl:      ctrl,
	}
}

func TestValidateSigned(t *testing.T) {
	t.Run("Test all signed commands basic - success", testValidateCommandsSuccess)
	t.Run("Test all signed commands basic - failure", testValidateCommandsFail)
	t.Run("Test submit order validations - success", testSubmitOrderValidationSuccess)
	t.Run("Test submit order validation - failure", testSubmitOrderValidationFail)
	t.Run("Test validate signed invalid payload", testValidateSignedInvalidPayload)
	t.Run("Test validate signed - invalid command", testValidateSignedInvalidCommand)
}

func TestProcess(t *testing.T) {
	t.Run("Test all basic process commands - Success", testProcessCommandSuccess)
}

func testValidateSignedInvalidPayload(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	party := []byte("party-id")
	cmd := blockchain.VoteCommand
	// wrong type for this command
	payload, err := proto.Marshal(&types.Proposal{
		PartyID: hex.EncodeToString(party),
	})
	assert.NoError(t, err)
	err = proc.ValidateSigned(party, payload, cmd)
	assert.Error(t, err)
}

func testValidateSignedInvalidCommand(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	var b byte // nil value
	assert.Error(t, proc.ValidateSigned([]byte("party"), []byte("foobar"), blockchain.Command(b)))
}

func testValidateCommandsFail(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString([]byte("another-party"))
	data := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		blockchain.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		blockchain.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		blockchain.ProposeCommand: &types.Proposal{
			PartyID: party,
		},
		blockchain.VoteCommand: &types.Vote{
			PartyID: party,
		},
		blockchain.WithdrawCommand: &types.Withdraw{
			PartyID: party,
		},
	}
	expError := map[blockchain.Command]error{
		blockchain.SubmitOrderCommand: processor.ErrOrderSubmissionPartyAndPubKeyDoesNotMatch,
		blockchain.CancelOrderCommand: processor.ErrOrderCancellationPartyAndPubKeyDoesNotMatch,
		blockchain.AmendOrderCommand:  processor.ErrOrderAmendmentPartyAndPubKeyDoesNotMatch,
		blockchain.ProposeCommand:     processor.ErrProposalSubmissionPartyAndPubKeyDoesNotMatch,
		blockchain.VoteCommand:        processor.ErrVoteSubmissionPartyAndPubKeyDoesNotMatch,
		blockchain.WithdrawCommand:    processor.ErrWithdrawPartyAndPublKeyDoesNotMatch,
	}
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	for cmd, msg := range data {
		payload, err := proto.Marshal(msg)
		assert.NoError(t, err)
		err = proc.ValidateSigned(key, payload, cmd)
		assert.Error(t, err)
		expErr, ok := expError[cmd]
		assert.True(t, ok)
		assert.Equal(t, expErr, err)
	}
}

func testValidateCommandsSuccess(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	data := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		blockchain.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		blockchain.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		blockchain.ProposeCommand: &types.Proposal{
			PartyID: party,
		},
		blockchain.VoteCommand: &types.Vote{
			PartyID: party,
		},
		blockchain.WithdrawCommand: &types.Withdraw{
			PartyID: party,
		},
	}
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	for cmd, msg := range data {
		payload, err := proto.Marshal(msg)
		assert.NoError(t, err)
		assert.NoError(t, proc.ValidateSigned(key, payload, cmd), "Failed to validate %v command payload", cmd)
	}
}

func testSubmitOrderValidationSuccess(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	party := []byte("party-id")
	// bare bones
	sub := &types.OrderSubmission{
		MarketID: "market-id",
		PartyID:  hex.EncodeToString(party),
		Price:    1,
		Size:     1,
	}
	payload, err := proto.Marshal(sub)
	assert.NoError(t, err)
	assert.NoError(t, proc.ValidateSigned(party, payload, blockchain.SubmitOrderCommand))
}

func testSubmitOrderValidationFail(t *testing.T) {
	proc := getTestProcessor(t)
	defer proc.ctrl.Finish()
	// different party
	party := []byte("other-party")
	sub := &types.OrderSubmission{
		MarketID: "market-id",
		PartyID:  hex.EncodeToString([]byte("party-id")),
		Price:    1,
		Size:     1,
	}
	payload, err := proto.Marshal(sub)
	assert.NoError(t, err)
	err = proc.ValidateSigned(party, payload, blockchain.SubmitOrderCommand)
	assert.Error(t, err)
	assert.Equal(t, err, processor.ErrOrderSubmissionPartyAndPubKeyDoesNotMatch)
}

func testProcessCommandSuccess(t *testing.T) {
	key := []byte("party-id")
	party := hex.EncodeToString(key)
	data := map[blockchain.Command]proto.Message{
		blockchain.SubmitOrderCommand: &types.OrderSubmission{
			PartyID: party,
		},
		blockchain.CancelOrderCommand: &types.OrderCancellation{
			PartyID: party,
		},
		blockchain.AmendOrderCommand: &types.OrderAmendment{
			PartyID: party,
		},
		blockchain.ProposeCommand: &types.Proposal{
			PartyID: party,
			Terms:   &types.ProposalTerms{}, // avoid nil bit, shouldn't be asset
		},
		blockchain.VoteCommand: &types.Vote{
			PartyID: party,
		},
		blockchain.WithdrawCommand: &types.Withdraw{
			PartyID: party,
		},
		blockchain.NotifyTraderAccountCommand: &types.NotifyTraderAccount{
			TraderID: party,
		},
	}
	proc := getTestProcessor(t)
	proc.eng.EXPECT().Withdraw(gomock.Any()).Times(1).Return(nil)
	proc.eng.EXPECT().SubmitOrder(gomock.Any()).Times(1).Return(&types.OrderConfirmation{}, nil)
	proc.eng.EXPECT().CancelOrder(gomock.Any()).Times(1).Return(&types.OrderCancellationConfirmation{}, nil)
	proc.eng.EXPECT().AmendOrder(gomock.Any()).Times(1).Return(&types.OrderConfirmation{}, nil)
	proc.eng.EXPECT().VoteOnProposal(gomock.Any()).Times(1).Return(nil)
	proc.eng.EXPECT().SubmitProposal(gomock.Any()).Times(1).Return(nil)
	proc.eng.EXPECT().NotifyTraderAccount(gomock.Any()).Times(1).Return(nil)
	defer proc.ctrl.Finish()
	for cmd, msg := range data {
		payload, err := proto.Marshal(msg)
		assert.NoError(t, err)
		assert.NoError(t, proc.Process(payload, cmd), "Failed to process %v command payload", cmd)
	}
}
