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
	"errors"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type DateRange struct {
	Start *time.Time
	End   *time.Time
}

var (
	ErrInvalidDateRange   = errors.New("invalid date range, date range is required")
	ErrMinimumDate        = errors.New("date range start must be after 2020-01-01")
	ErrEndDateBeforeStart = errors.New("date range start must be before end")
	ErrDateRangeTooLong   = errors.New("date range is too long")
	minimumDate           = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	maximumDuration       = time.Hour * 24 * 365 // 1 year maximum duration
)

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

func (dr DateRange) Validate(required bool) error {
	if !required && dr.Start == nil && dr.End == nil {
		return nil
	}

	if required && dr.Start == nil && dr.End == nil {
		return ErrInvalidDateRange
	}

	if dr.Start != nil && dr.Start.Before(minimumDate) {
		return ErrMinimumDate
	}

	if dr.End != nil && dr.End.Before(minimumDate) {
		return ErrMinimumDate
	}

	if dr.Start != nil && dr.End != nil && dr.Start.After(*dr.End) {
		return ErrEndDateBeforeStart
	}

	end := time.Now()
	if dr.End != nil {
		end = *dr.End
	}

	start := minimumDate
	if dr.Start != nil {
		start = *dr.Start
	}

	if end.Sub(start) > maximumDuration {
		return ErrDateRangeTooLong
	}

	return nil
}
