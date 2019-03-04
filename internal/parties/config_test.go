package parties

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"vega/internal/logging"
)

func TestConfig_GetLogger(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	partyConfig := NewDefaultConfig(logger)
	l := partyConfig.GetLogger()

	assert.Equal(t, namedLogger, l.GetName())
}

func TestConfig_UpdateLogger(t *testing.T) {
	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	partyConfig := NewDefaultConfig(logger)
	partyConfig.UpdateLogger()

	l := partyConfig.GetLogger()
	assert.Equal(t, logging.InfoLevel, partyConfig.Level)
	assert.Equal(t, logging.InfoLevel, l.GetLevel())

	partyConfig.Level = logging.DebugLevel
	partyConfig.UpdateLogger()

	assert.Equal(t, logging.DebugLevel, partyConfig.Level)
	assert.Equal(t, logging.DebugLevel, l.GetLevel())

}
