package node_test

import (
	"testing"

	"code.vegaprotocol.io/vega/wallet/api/node"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	// Change the level to debug for debugging.
	// Keep it to Panic otherwise to not pollute tests output.
	return zaptest.NewLogger(t, zaptest.Level(zap.PanicLevel))
}

func noReporting(_ node.ReportType, _ string) {
	// Nothing to do.
}
