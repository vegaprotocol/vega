// Copyright (C) 2023  Gobalsky Labs Limited
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

//go:build windows

package zap

import (
	"fmt"
	"net/url"
	"os"

	"go.uber.org/zap"
)

func init() {
	if err := zap.RegisterSink("winfile", newWinFileSink); err != nil {
		panic(fmt.Errorf("couldn't register the windows file sink: %w", err))
	}
}

func toOSFilePath(logPath string) string {
	return "winfile:///" + logPath
}

// newWinFileSink creates a log sink on Windows machines as zap, by default,
// doesn't support Windows paths. A workaround is to create a fake winfile
// scheme and register it with zap instead. This workaround is taken from
// the GitHub issue at https://github.com/uber-go/zap/issues/621.
func newWinFileSink(u *url.URL) (zap.Sink, error) {
	// Remove leading slash left by url.Parse().
	return os.OpenFile(u.Path[1:], os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644) //nolint:gomnd
}
