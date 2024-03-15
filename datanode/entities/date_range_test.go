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

package entities_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/ptr"
)

func TestDateRange_Validate(t *testing.T) {
	type args struct {
		Start    *time.Time
		End      *time.Time
		Required bool
	}
	tests := []struct {
		name string
		args args
		Err  error
	}{
		{
			name: "Should error if required and no dates provided",
			args: args{
				Start:    nil,
				End:      nil,
				Required: true,
			},
			Err: entities.ErrInvalidDateRange,
		},
		{
			name: "Should not error if not required and no dates provided",
			args: args{
				Start:    nil,
				End:      nil,
				Required: false,
			},
			Err: nil,
		},
		{
			name: "Should error if start date is before minimum date, required false",
			args: args{
				Start:    ptr.From(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)),
				End:      nil,
				Required: false,
			},
			Err: entities.ErrMinimumDate,
		},
		{
			name: "Should error if start date is before minimum date, required true",
			args: args{
				Start:    ptr.From(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)),
				End:      nil,
				Required: true,
			},
			Err: entities.ErrMinimumDate,
		},
		{
			name: "Should error if end date is before minimum date, required false",
			args: args{
				Start:    nil,
				End:      ptr.From(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)),
				Required: false,
			},
			Err: entities.ErrMinimumDate,
		},
		{
			name: "Should error if end date is before minimum date, required true",
			args: args{
				Start:    nil,
				End:      ptr.From(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)),
				Required: true,
			},
			Err: entities.ErrMinimumDate,
		},
		{
			name: "Should error if start and end date is before minimum date, required false",
			args: args{
				Start:    ptr.From(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)),
				End:      ptr.From(time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC)),
				Required: false,
			},
			Err: entities.ErrMinimumDate,
		},
		{
			name: "Should error if start and end date is before minimum date, required true",
			args: args{
				Start:    ptr.From(time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)),
				End:      ptr.From(time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC)),
				Required: true,
			},
			Err: entities.ErrMinimumDate,
		},
		{
			name: "Should error if end date is before start date, required false",
			args: args{
				Start:    ptr.From(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				End:      ptr.From(time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)),
				Required: false,
			},
			Err: entities.ErrEndDateBeforeStart,
		},
		{
			name: "Should error if end date is before start date, required true",
			args: args{
				Start:    ptr.From(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				End:      ptr.From(time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)),
				Required: true,
			},
			Err: entities.ErrEndDateBeforeStart,
		},
		{
			name: "Should error if duration is more than max, required false",
			args: args{
				Start:    ptr.From(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
				End:      ptr.From(time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)),
				Required: false,
			},
			Err: entities.ErrDateRangeTooLong,
		},
		{
			name: "Should error if duration is more than max, required true",
			args: args{
				Start:    ptr.From(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
				End:      ptr.From(time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)),
				Required: true,
			},
			Err: entities.ErrDateRangeTooLong,
		},
		{
			name: "Should error if duration is more than max, no start date",
			args: args{
				Start:    nil,
				End:      ptr.From(time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)),
				Required: true,
			},
			Err: entities.ErrDateRangeTooLong,
		},
		{
			name: "Should error if duration is more than max, no end date",
			args: args{
				Start:    ptr.From(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				End:      nil,
				Required: true,
			},
			Err: entities.ErrDateRangeTooLong,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := entities.DateRange{
				Start: tt.args.Start,
				End:   tt.args.End,
			}
			if err := dr.Validate(tt.args.Required); err != tt.Err {
				t.Errorf("DateRange.Validate() error = %v, wantErr %v", err, tt.Err)
			}
		})
	}
}
