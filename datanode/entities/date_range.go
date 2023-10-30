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

package entities

import (
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type DateRange struct {
	Start *time.Time
	End   *time.Time
}

func DateRangeFromProto(dateRangeInput *v2.DateRange) (dateRange DateRange) {
	if dateRangeInput == nil {
		return
	}

	if dateRangeInput.StartTimestamp != nil {
		dateRange.Start = ptr.From(time.Unix(0, *dateRangeInput.StartTimestamp))
	}

	if dateRangeInput.EndTimestamp != nil {
		dateRange.End = ptr.From(time.Unix(0, *dateRangeInput.EndTimestamp))
	}

	return
}
