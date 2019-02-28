package blockchain

import (
	"testing"

	execution "vega/internal/execution/mocks"
	"vega/internal/logging"
	"vega/internal/vegatime"

	"github.com/stretchr/testify/assert"
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
