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

	"go.uber.org/zap"
)

func Build(cfg zap.Config) (*zap.Logger, error) {
	log, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("couldn't create logger: %w", err)
	}
	return log, nil
}

func BuildJSONFileLogger(level, filePath string) (*zap.Logger, error) {
	return Build(WithJSONFormat(WithFileOutput(WithLevel(DefaultConfig(), level), filePath)))
}

func BuildStandardConsoleLogger(level string) (*zap.Logger, error) {
	return Build(WithStandardOutput(WithConsoleFormat(WithLevel(DefaultConfig(), level))))
}

func BuildStandardJSONLogger(level string) (*zap.Logger, error) {
	return Build(WithStandardOutput(WithJSONFormat(WithLevel(DefaultConfig(), level))))
}
