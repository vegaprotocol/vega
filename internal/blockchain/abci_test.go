package blockchain

import (
	"testing"
	"vega/internal/vegatime"
	execution "vega/internal/execution/mocks"
	"github.com/stretchr/testify/assert"
	"vega/internal/logging"
)

func TestNewAbciApplication(t *testing.T) {
	ex := &execution.Engine{}
	vt := vegatime.NewTimeService(nil)

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	config := NewConfig(logger)
	stats := NewStats()
	chain := NewAbciApplication(config, stats, ex, vt)
	assert.Equal(t, uint64(0), chain.height)
}
