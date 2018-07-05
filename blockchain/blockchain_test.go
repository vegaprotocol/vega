package blockchain

import (
	"testing"
	"vega/core"

	"github.com/stretchr/testify/assert"
)

func TestNewBlockchain(t *testing.T) {
	config := core.Config{}
	vegaApp := core.New(config)
	chain := NewBlockchain(vegaApp)

	assert.Equal(t, chain.state.Height, int64(0))
}
