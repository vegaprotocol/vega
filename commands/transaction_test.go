package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckTransaction(t *testing.T) {
	t.Run("Empty transaction should fail", testEmptyTransactionShouldFail)
}

func testEmptyTransactionShouldFail(t *testing.T) {
	err := commands.CheckTransaction(&commandspb.Transaction{})
	assert.EqualError(t, err, "tx.from (is required), tx.input_data (is required), tx.signature (is required)")
}
