package snapshot_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/snapshot"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/require"
)

func TestEngineConfig(t *testing.T) {
	t.Run("Default configuration is valid", testEngineConfigDefaultConfigIsValid)
	t.Run("Invalid configuration fails", testEngineConfigInvalidConfigFails)
}

func testEngineConfigDefaultConfigIsValid(t *testing.T) {
	defaultConfig := snapshot.DefaultConfig()

	require.NoError(t, defaultConfig.Validate())
}

func testEngineConfigInvalidConfigFails(t *testing.T) {
	// StartHeight
	defaultConfig := snapshot.DefaultConfig()

	defaultConfig.StartHeight = -1

	require.Error(t, defaultConfig.Validate())

	// KeepRecent
	defaultConfig = snapshot.DefaultConfig()

	defaultConfig.KeepRecent = 0

	require.Error(t, defaultConfig.Validate())

	// Storage
	defaultConfig = snapshot.DefaultConfig()

	defaultConfig.Storage = vgrand.RandomStr(3)
	require.Error(t, defaultConfig.Validate())
}
