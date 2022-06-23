package node_test

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	// Change the level to debug for debugging.
	// Keep it to Panic otherwise to not pollute tests output.
	return zaptest.NewLogger(t, zaptest.Level(zap.PanicLevel))
}
