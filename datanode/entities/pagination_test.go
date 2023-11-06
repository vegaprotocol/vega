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

package entities_test

import (
	"encoding/base64"
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/stretchr/testify/assert"
)

func TestCursorPaginationFromProto_Combinations(t *testing.T) {
	// returns default when pagination is nil
	cursor, err := entities.CursorPaginationFromProto(nil)
	assert.NoError(t, err)
	assert.Equal(t, entities.DefaultCursorPagination(true), cursor)

	// returns default but uses newest first variable when defined
	pagination := &v2.Pagination{NewestFirst: ptr.From(false)}
	cursor, err = entities.CursorPaginationFromProto(pagination)
	assert.NoError(t, err)
	assert.Equal(t, entities.DefaultCursorPagination(false), cursor)

	// uses the first filter
	pagination = &v2.Pagination{First: ptr.From[int32](100)}
	cursor, err = entities.CursorPaginationFromProto(pagination)
	assert.NoError(t, err)
	assert.Equal(t, entities.CursorPagination{
		NewestFirst: true,
		Forward:     &entities.CursorOffset{Limit: ptr.From[int32](100)},
	}, cursor)

	// uses the last filter
	pagination = &v2.Pagination{Last: ptr.From[int32](50)}
	cursor, err = entities.CursorPaginationFromProto(pagination)
	assert.NoError(t, err)
	assert.Equal(t, entities.CursorPagination{
		NewestFirst: true,
		Backward:    &entities.CursorOffset{Limit: ptr.From[int32](50)},
	}, cursor)

	encodedTestCursor := base64.StdEncoding.EncodeToString([]byte("abcd"))
	testCursor := &entities.Cursor{}
	testCursor.Decode(encodedTestCursor)
	assert.NoError(t, err)

	defaultPageSize := ptr.From[int32](entities.DefaultPageSize)

	// uses after filter
	pagination = &v2.Pagination{After: ptr.From(encodedTestCursor)}
	cursor, err = entities.CursorPaginationFromProto(pagination)
	assert.NoError(t, err)
	assert.Equal(t, entities.CursorPagination{
		NewestFirst: true,
		Forward: &entities.CursorOffset{
			Limit:  defaultPageSize,
			Cursor: testCursor,
		},
	}, cursor)

	// uses before filter
	pagination = &v2.Pagination{Before: ptr.From(encodedTestCursor)}
	cursor, err = entities.CursorPaginationFromProto(pagination)
	assert.NoError(t, err)
	assert.Equal(t, entities.CursorPagination{
		NewestFirst: true,
		Backward: &entities.CursorOffset{
			Limit:  defaultPageSize,
			Cursor: testCursor,
		},
	}, cursor)

	// uses the first and after filters in conjunction
	pagination = &v2.Pagination{First: ptr.From[int32](200), After: ptr.From(encodedTestCursor)}
	cursor, err = entities.CursorPaginationFromProto(pagination)
	assert.NoError(t, err)
	assert.Equal(t, entities.CursorPagination{
		NewestFirst: true,
		Forward: &entities.CursorOffset{
			Limit:  ptr.From[int32](200),
			Cursor: testCursor,
		},
	}, cursor)

	// uses the last and before filters in conjunction
	pagination = &v2.Pagination{Last: ptr.From[int32](200), Before: ptr.From(encodedTestCursor)}
	cursor, err = entities.CursorPaginationFromProto(pagination)
	assert.NoError(t, err)
	assert.Equal(t, entities.CursorPagination{
		NewestFirst: true,
		Backward: &entities.CursorOffset{
			Limit:  ptr.From[int32](200),
			Cursor: testCursor,
		},
	}, cursor)

	// using first and before ignores before
	pagination = &v2.Pagination{First: ptr.From[int32](30), Before: ptr.From(encodedTestCursor)}
	cursor, err = entities.CursorPaginationFromProto(pagination)
	assert.NoError(t, err)
	assert.Equal(t, entities.CursorPagination{
		NewestFirst: true,
		Forward: &entities.CursorOffset{
			Limit: ptr.From[int32](30),
		},
	}, cursor)

	// using last and after ignores after
	pagination = &v2.Pagination{Last: ptr.From[int32](50), After: ptr.From(encodedTestCursor)}
	cursor, err = entities.CursorPaginationFromProto(pagination)
	assert.NoError(t, err)
	assert.Equal(t, entities.CursorPagination{
		NewestFirst: true,
		Backward: &entities.CursorOffset{
			Limit: ptr.From[int32](50),
		},
	}, cursor)
}
