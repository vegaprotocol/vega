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

package gql

import (
	"fmt"
	"math"
	"strconv"

	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/libs/ptr"
)

func safeStringUint64(input string) (uint64, error) {
	i, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		// A conversion error occurred, return the error
		return 0, fmt.Errorf("invalid input string for uint64 conversion %s", input)
	}
	return i, nil
}

func safeStringInt64(input string) (int64, error) {
	i, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		// A conversion error occurred, return the error
		return 0, fmt.Errorf("invalid input string for int64 conversion %s", input)
	}
	return i, nil
}

func secondsTSToDatetime(timestampInSeconds int64) string {
	return vegatime.Format(vegatime.Unix(timestampInSeconds, 0))
}

func nanoTSToDatetime(timestampInNanoSeconds int64) string {
	return vegatime.Format(vegatime.UnixNano(timestampInNanoSeconds))
}

func convertVersion(version *int) (*int32, error) {
	if version != nil {
		if *version >= 1 && *version < math.MaxInt32 {
			return ptr.From(int32(*version)), nil
		}
		return nil, fmt.Errorf("invalid version value %d", *version)
	}
	return nil, nil
}
