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

package zap

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var SupportedLogLevels = []string{
	zapcore.DebugLevel.String(),
	zapcore.InfoLevel.String(),
	zapcore.WarnLevel.String(),
	zapcore.ErrorLevel.String(),
}

func IsSupportedLogLevel(level string) bool {
	for _, supported := range SupportedLogLevels {
		if level == supported {
			return true
		}
	}
	return false
}

func EnsureIsSupportedLogLevel(level string) error {
	if !IsSupportedLogLevel(level) {
		return fmt.Errorf("unsupported log level %q, supported levels: %s", level, strings.Join(SupportedLogLevels, ", "))
	}
	return nil
}

func parseLevel(level string) (zap.AtomicLevel, error) {
	if err := EnsureIsSupportedLogLevel(level); err != nil {
		return zap.AtomicLevel{}, err
	}

	l := new(zapcore.Level)

	if err := l.UnmarshalText([]byte(level)); err != nil {
		return zap.AtomicLevel{}, fmt.Errorf("couldn't parse log level: %w", err)
	}

	return zap.NewAtomicLevelAt(*l), nil
}
