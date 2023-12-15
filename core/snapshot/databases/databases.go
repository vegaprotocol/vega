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

package databases

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/paths"
)

func RemoveAll(vegaPaths paths.Paths) error {
	dbDirectory := vegaPaths.StatePathFor(paths.SnapshotStateHome)

	if err := os.RemoveAll(dbDirectory); err != nil {
		return fmt.Errorf("an error occurred while removing directory %q: %w", dbDirectory, err)
	}

	return nil
}
