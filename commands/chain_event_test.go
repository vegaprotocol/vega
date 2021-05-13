package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckChainEvent(t *testing.T) {
	t.Run("Submitting a chain event without tx ID fails", testChainEventWithoutTxIDFails)
	t.Run("Submitting a chain event with tx ID succeeds", testChainEventWithTxIDSucceeds)
	t.Run("Submitting a chain event without nonce fails", testChainEventWithoutNonceFails)
	t.Run("Submitting a chain event with nonce succeeds", testChainEventWithNonceSucceeds)
}

func testChainEventWithoutTxIDFails(t *testing.T) {
	err := checkChainEvent(&commandspb.ChainEvent{})
	assert.Contains(t, err.Get("chain_event.tx_id"), commands.ErrIsRequired)
}

func testChainEventWithTxIDSucceeds(t *testing.T) {
	err := checkChainEvent(&commandspb.ChainEvent{
		TxId: "my ID",
	})
	assert.NotContains(t, err.Get("chain_event.tx_id"), commands.ErrIsRequired)
}

func testChainEventWithoutNonceFails(t *testing.T) {
	err := checkChainEvent(&commandspb.ChainEvent{})
	assert.Contains(t, err.Get("chain_event.nonce"), commands.ErrIsRequired)
}

func testChainEventWithNonceSucceeds(t *testing.T) {
	err := checkChainEvent(&commandspb.ChainEvent{
		Nonce: RandomPositiveU64(),
	})
	assert.NotContains(t, err.Get("chain_event.nonce"), commands.ErrIsRequired)
}

func checkChainEvent(cmd *commandspb.ChainEvent) commands.Errors {
	err := commands.CheckChainEvent(cmd)

	e, ok := err.(commands.Errors)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
