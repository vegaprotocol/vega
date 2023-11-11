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

	types "code.vegaprotocol.io/vega/protos/vega"
)

func convertDataNodeIntervalToProto(interval string) (types.Interval, error) {
	switch interval {
	case "block":
		return types.Interval_INTERVAL_BLOCK, nil
	case "1 minute":
		return types.Interval_INTERVAL_I1M, nil
	case "5 minutes":
		return types.Interval_INTERVAL_I5M, nil
	case "15 minutes":
		return types.Interval_INTERVAL_I15M, nil
	case "1 hour":
		return types.Interval_INTERVAL_I1H, nil
	case "6 hours":
		return types.Interval_INTERVAL_I6H, nil
	case "1 day":
		return types.Interval_INTERVAL_I1D, nil
	default:
		err := fmt.Errorf("failed to convert Interval from GraphQL to Proto: %v", interval)
		return types.Interval_INTERVAL_UNSPECIFIED, err
	}
}
