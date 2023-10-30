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

package zap

import (
	"fmt"
)

type Logger interface {
	Sync() error
}

func Sync(logger Logger) func() {
	return func() {
		err := logger.Sync()
		if err != nil {
			// This is the ultimate warning, as we can't do anything else.
			fmt.Printf("couldn't flush the logger: %v", err) //nolint:forbidigo
		}
	}
}
