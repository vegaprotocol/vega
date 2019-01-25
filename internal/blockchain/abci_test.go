package blockchain

import (
	"testing"
	"github.com/stretchr/testify/assert"
    execution "vega/internal/execution/mocks"
	"vega/vegatime"
)

func TestNewAbciApplication(t *testing.T) {

	ex := &execution.Engine{}
	vt := vegatime.NewVegaTimeService(nil)
	config := NewConfig()
	stats := NewStats()
	
	chain := NewAbciApplication(config, ex, vt, stats)

	assert.Equal(t, uint64(0), chain.height)
}

