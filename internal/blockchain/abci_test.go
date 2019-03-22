package blockchain

import (
	"testing"

	execution "code.vegaprotocol.io/vega/internal/execution/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/stretchr/testify/assert"
)

func TestNewAbciApplication(t *testing.T) {
	ex := &execution.Engine{}
	vt := vegatime.NewTimeService(nil)

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	config := NewDefaultConfig(logger)
	stats := NewStats()
	chain := NewAbciApplication(config, stats, ex, vt, func() {})
	assert.Equal(t, uint64(0), chain.height)
}
