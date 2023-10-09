// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
