package tools

import (
	"testing"

	"code.vegaprotocol.io/vega/vegatools/checktx"

	"github.com/stretchr/testify/assert"
)

func TestTxReturnsNoErrorWhenCheckingCompatibleTransaction(t *testing.T) {
	encodedTransaction, err := checktx.CreatedEncodedTransactionData()
	assert.NoErrorf(t, err, "error was returned when creating test data\nerr: %v", err)

	cmd := checkTxCmd{
		EncodedTransaction: encodedTransaction,
	}

	err = cmd.Execute(nil)
	assert.NoErrorf(t, err, "error was returned when the transaction should have been valid")
}

func TestTxReturnsErrorWhenCheckingIncompatibleTransaction(t *testing.T) {
	cmd := checkTxCmd{
		EncodedTransaction: "12345",
	}

	err := cmd.Execute(nil)
	assert.Error(t, err, "")
}
