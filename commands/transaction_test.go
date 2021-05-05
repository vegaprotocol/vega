package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckTransaction(t *testing.T) {
	t.Run("check empty transaction", testEmptyTransaction)
}

func testEmptyTransaction(t *testing.T) {
	err := commands.CheckTransaction(&commandspb.Transaction{})
	assert.EqualError(t, err, "tx.input_data is required, tx.signature is required, tx.from is required")
}
