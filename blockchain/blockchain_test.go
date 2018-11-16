package blockchain

import (
	"testing"
	"vega/core"

	"github.com/stretchr/testify/assert"
	"vega/datastore/mocks"
)

func TestNewBlockchain(t *testing.T) {
	config := core.GetConfig()

	// Vega core
	vega := core.New(config, &mocks.OrderStore{}, &mocks.TradeStore{}, &mocks.CandleStore{})
	chain := NewBlockchain(vega)

	assert.Equal(t, chain.vega.State.Height, int64(0))
}
