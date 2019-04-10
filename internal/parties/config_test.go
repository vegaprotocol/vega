package parties_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/parties"
	"github.com/stretchr/testify/assert"
)

func TestConfig_GetLogger(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()

	partyConfig := parties.NewDefaultConfig(logger)
	l := partyConfig.GetLogger()

	assert.Equal(t, parties.NamedLogger, l.GetName())
}

func TestConfig_UpdateLogger(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()

	partyConfig := parties.NewDefaultConfig(logger)
	partyConfig.UpdateLogger()

	l := partyConfig.GetLogger()
	assert.Equal(t, logging.InfoLevel, partyConfig.Level)
	assert.Equal(t, logging.InfoLevel, l.GetLevel())

	partyConfig.Level = logging.DebugLevel
	partyConfig.UpdateLogger()

	assert.Equal(t, logging.DebugLevel, partyConfig.Level)
	assert.Equal(t, logging.DebugLevel, l.GetLevel())

}
