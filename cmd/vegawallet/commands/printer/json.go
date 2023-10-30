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

package printer

import (
	"encoding/json"
	"fmt"
	"io"
)

func FprintJSON(w io.Writer, data interface{}) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("unable to marshal message: %w", err)
	}

	if _, err = fmt.Fprintf(w, "%v\n", string(buf)); err != nil {
		return fmt.Errorf("couldn't print data to %v: %w", w, err)
	}

	return nil
}
