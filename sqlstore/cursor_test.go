package sqlstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCursor_Where(t *testing.T) {
	type args struct {
		Cmp  string
		args []interface{}
	}

	testCases := []struct {
		name      string
		cursor    CursorQueryParameter
		args      args
		wantWhere string
		wantArgs  []interface{}
	}{
		{
			name: "Equal",
			cursor: CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       ASC,
				Cmp:        EQ,
				Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
			},
			args: args{
				args: nil,
			},
			wantWhere: "vega_time = $1",
			wantArgs:  []interface{}{time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC)},
		},
		{
			name: "Less than or equal",
			cursor: CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       ASC,
				Cmp:        LE,
				Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
			},
			args: args{
				args: []interface{}{"TEST"},
			},
			wantWhere: "vega_time <= $2",
			wantArgs:  []interface{}{"TEST", time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC)},
		},
		{
			name: "Greater than or equal",
			cursor: CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       ASC,
				Cmp:        GE,
				Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
			},
			args: args{
				args: nil,
			},
			wantWhere: "vega_time >= $1",
			wantArgs:  []interface{}{time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			gotWhere, gotArgs := tc.cursor.Where(tc.args.args...)
			assert.Equal(t, tc.wantWhere, gotWhere)
			assert.Equal(t, tc.wantArgs, gotArgs)
		})
	}
}

func TestCursor_OrderBy(t *testing.T) {
	testCases := []struct {
		name      string
		cursor    CursorQueryParameter
		wantOrder string
	}{
		{
			name: "Ascending",
			cursor: CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       ASC,
				Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
			},
			wantOrder: "vega_time ASC",
		},
		{
			name: "Descending",
			cursor: CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       DESC,
				Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
			},
			wantOrder: "vega_time DESC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			got := tc.cursor.OrderBy()
			assert.Equal(t, tc.wantOrder, got)
		})
	}
}

func TestCursors_Where(t *testing.T) {
	testCases := []struct {
		name      string
		cursors   CursorQueryParameters
		wantWhere string
		wantArgs  []interface{}
	}{
		{
			name: "One cursor",
			cursors: CursorQueryParameters{
				CursorQueryParameter{
					ColumnName: "vega_time",
					Sort:       ASC,
					Cmp:        EQ,
					Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
				},
			},
			wantWhere: "vega_time = $1",
			wantArgs:  []interface{}{time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC)},
		},
		{
			name: "Two cursors",
			cursors: CursorQueryParameters{
				CursorQueryParameter{
					ColumnName: "vega_time",
					Sort:       ASC,
					Cmp:        EQ,
					Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
				},
				{
					ColumnName: "seq_num",
					Sort:       ASC,
					Cmp:        GE,
					Value:      1,
				},
			},
			wantWhere: "vega_time = $1 AND seq_num >= $2",
			wantArgs:  []interface{}{time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC), 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			gotWhere, gotArgs := tc.cursors.Where()
			assert.Equal(t, tc.wantWhere, gotWhere)
			assert.Equal(t, tc.wantArgs, gotArgs)
		})
	}
}

func TestCursors_OrderBy(t *testing.T) {
	testCases := []struct {
		name      string
		cursors   CursorQueryParameters
		wantOrder string
	}{
		{
			name: "One cursor",
			cursors: CursorQueryParameters{
				CursorQueryParameter{
					ColumnName: "vega_time",
					Sort:       ASC,
					Cmp:        EQ,
					Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
				},
			},
			wantOrder: "vega_time ASC",
		},
		{
			name: "Two cursors",
			cursors: CursorQueryParameters{
				{
					ColumnName: "vega_time",
					Sort:       ASC,
					Cmp:        EQ,
					Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
				},
				{
					ColumnName: "seq_num",
					Sort:       ASC,
					Cmp:        GE,
					Value:      1,
				},
			},
			wantOrder: "vega_time ASC, seq_num ASC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			got := tc.cursors.OrderBy()
			assert.Equal(t, tc.wantOrder, got)
		})
	}
}
