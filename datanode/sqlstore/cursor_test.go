// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorPredicate(t *testing.T) {
	type Cursor struct {
		Foo int
		Bar int `db:"baz"`
	}
	cursor := Cursor{Foo: 1, Bar: 2}

	testCases := []struct {
		name              string
		ordering          sqlstore.TableOrdering
		expectedPredicate string
		expectedArgs      []interface{}
	}{
		{
			name: "Single",
			ordering: sqlstore.TableOrdering{
				sqlstore.NewColumnOrdering("foo", sqlstore.ASC),
			},
			expectedPredicate: "(foo >= $1)",
			expectedArgs:      []any{1},
		},
		{
			name: "Reversed",
			ordering: sqlstore.TableOrdering{
				sqlstore.NewColumnOrdering("foo", sqlstore.DESC),
			},
			expectedPredicate: "(foo <= $1)",
			expectedArgs:      []any{1},
		},
		{
			name: "Composite",
			ordering: sqlstore.TableOrdering{
				sqlstore.NewColumnOrdering("foo", sqlstore.ASC),
				sqlstore.NewColumnOrdering("baz", sqlstore.DESC),
			},
			expectedPredicate: "(foo > $1) OR (foo = $1 AND baz <= $2)",
			expectedArgs:      []any{1, 2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			predicate, args, err := sqlstore.CursorPredicate(nil, cursor, tc.ordering)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedPredicate, predicate)
			assert.Equal(t, tc.expectedArgs, args)
		})
	}
}

func TestCursor_Where(t *testing.T) {
	type args struct {
		Cmp  string
		args []interface{}
	}

	testCases := []struct {
		name      string
		cursor    sqlstore.CursorQueryParameter
		args      args
		wantWhere string
		wantArgs  []interface{}
	}{
		{
			name: "Equal",
			cursor: sqlstore.CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       sqlstore.ASC,
				Cmp:        sqlstore.EQ,
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
			cursor: sqlstore.CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       sqlstore.ASC,
				Cmp:        sqlstore.LE,
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
			cursor: sqlstore.CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       sqlstore.ASC,
				Cmp:        sqlstore.GE,
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
		cursor    sqlstore.CursorQueryParameter
		wantOrder string
	}{
		{
			name: "Ascending",
			cursor: sqlstore.CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       sqlstore.ASC,
				Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
			},
			wantOrder: "vega_time ASC",
		},
		{
			name: "Descending",
			cursor: sqlstore.CursorQueryParameter{
				ColumnName: "vega_time",
				Sort:       sqlstore.DESC,
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
		cursors   sqlstore.CursorQueryParameters
		wantWhere string
		wantArgs  []interface{}
	}{
		{
			name: "One cursor",
			cursors: sqlstore.CursorQueryParameters{
				{
					ColumnName: "vega_time",
					Sort:       sqlstore.ASC,
					Cmp:        sqlstore.EQ,
					Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
				},
			},
			wantWhere: "vega_time = $1",
			wantArgs:  []interface{}{time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC)},
		},
		{
			name: "Two cursors",
			cursors: sqlstore.CursorQueryParameters{
				{
					ColumnName: "vega_time",
					Sort:       sqlstore.ASC,
					Cmp:        sqlstore.EQ,
					Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
				},
				{
					ColumnName: "seq_num",
					Sort:       sqlstore.ASC,
					Cmp:        sqlstore.GE,
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
		cursors   sqlstore.CursorQueryParameters
		wantOrder string
	}{
		{
			name: "One cursor",
			cursors: sqlstore.CursorQueryParameters{
				sqlstore.CursorQueryParameter{
					ColumnName: "vega_time",
					Sort:       sqlstore.ASC,
					Cmp:        sqlstore.EQ,
					Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
				},
			},
			wantOrder: "vega_time ASC",
		},
		{
			name: "Two cursors",
			cursors: sqlstore.CursorQueryParameters{
				{
					ColumnName: "vega_time",
					Sort:       sqlstore.ASC,
					Cmp:        sqlstore.EQ,
					Value:      time.Date(2022, 5, 9, 9, 0, 0, 0, time.UTC),
				},
				{
					ColumnName: "seq_num",
					Sort:       sqlstore.ASC,
					Cmp:        sqlstore.GE,
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
