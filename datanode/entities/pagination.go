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

package entities

import (
	"encoding/base64"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/pkg/errors"
)

const (
	DefaultPageSize int32 = 1000
	maxPageSize     int32 = 5000
)

var ErrCursorOverflow = errors.Errorf("pagination limit must be in range 0-%d", maxPageSize)

type Pagination interface{}

type Cursor struct {
	cursor string
}

func NewCursor(cursor string) *Cursor {
	return &Cursor{
		cursor: cursor,
	}
}

func (c *Cursor) Encode() string {
	if c.cursor == "" {
		return ""
	}

	return base64.StdEncoding.EncodeToString([]byte(c.cursor))
}

func (c *Cursor) Decode(value string) error {
	if value == "" {
		return errors.New("cursor is empty")
	}

	cursor, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return errors.Wrap(err, "failed to decode cursor")
	}

	c.cursor = string(cursor)

	return nil
}

func (c *Cursor) IsSet() bool {
	return c.cursor != ""
}

func (c *Cursor) Value() string {
	return c.cursor
}

type CursorPagination struct {
	Pagination
	Forward     *CursorOffset
	Backward    *CursorOffset
	NewestFirst bool
}

func (p CursorPagination) HasForward() bool {
	return p.Forward != nil
}

func (p CursorPagination) HasBackward() bool {
	return p.Backward != nil
}

func NewCursorPagination(first *int32, after *string, last *int32, before *string, newestFirst bool) (CursorPagination, error) {
	return CursorPaginationFromProto(&v2.Pagination{
		First:       first,
		After:       after,
		Last:        last,
		Before:      before,
		NewestFirst: &newestFirst,
	})
}

func CursorPaginationFromProto(cp *v2.Pagination) (CursorPagination, error) {
	if cp == nil || (cp != nil && cp.First == nil && cp.Last == nil && cp.After == nil && cp.Before == nil) {
		if cp != nil && cp.NewestFirst != nil {
			return DefaultCursorPagination(*cp.NewestFirst), nil
		}
		return DefaultCursorPagination(true), nil
	}

	var after, before Cursor
	var err error
	var forwardOffset, backwardOffset *CursorOffset

	if cp.Before != nil && cp.After != nil {
		return CursorPagination{}, errors.New("cannot set both a before and after cursor")
	}

	if cp.First != nil {
		if *cp.First < 0 || *cp.First > maxPageSize {
			return CursorPagination{}, ErrCursorOverflow
		}
		forwardOffset = &CursorOffset{
			Limit: cp.First,
		}
		// Proto cursors should be encoded values, so we want to decode them in order to use them
		if cp.After != nil {
			if err = after.Decode(*cp.After); err != nil {
				return CursorPagination{}, errors.Wrap(err, "failed to decode after cursor")
			}

			forwardOffset.Cursor = &after
		}
	} else if cp.Last != nil {
		if *cp.Last < 0 || *cp.Last > maxPageSize {
			return CursorPagination{}, ErrCursorOverflow
		}
		backwardOffset = &CursorOffset{
			Limit: cp.Last,
		}
		// Proto cursors should be encoded values, so we want to decode them in order to use them
		if cp.Before != nil {
			if err = before.Decode(*cp.Before); err != nil {
				return CursorPagination{}, errors.Wrap(err, "failed to decode before cursor")
			}
			backwardOffset.Cursor = &before
		}
	} else if cp.After != nil {
		// Have an 'after' cursor but no page size ('first') so use default
		if err = after.Decode(*cp.After); err != nil {
			return CursorPagination{}, errors.Wrap(err, "failed to decode after cursor")
		}

		forwardOffset = &CursorOffset{
			Limit:  ptr.From(DefaultPageSize),
			Cursor: &after,
		}
	} else if cp.Before != nil {
		// Have an 'before' cursor but no page size ('first') so use default
		if err = before.Decode(*cp.Before); err != nil {
			return CursorPagination{}, errors.Wrap(err, "failed to decode before cursor")
		}

		backwardOffset = &CursorOffset{
			Limit:  ptr.From(DefaultPageSize),
			Cursor: &before,
		}
	}

	// Default the sort order to return the newest records first if no sort order is provided
	newestFirst := true
	if cp.NewestFirst != nil {
		newestFirst = *cp.NewestFirst
	}

	pagination := CursorPagination{
		Forward:     forwardOffset,
		Backward:    backwardOffset,
		NewestFirst: newestFirst,
	}

	if err = validatePagination(pagination); err != nil {
		return CursorPagination{}, err
	}

	return pagination, nil
}

func DefaultCursorPagination(newestFirst bool) CursorPagination {
	return CursorPagination{
		Forward: &CursorOffset{
			Limit: ptr.From(DefaultPageSize),
		},
		NewestFirst: newestFirst,
	}
}

type CursorOffset struct {
	Limit  *int32
	Cursor *Cursor
}

func (o CursorOffset) IsSet() bool {
	return o.Limit != nil
}

func (o CursorOffset) HasCursor() bool {
	return o.Cursor != nil && o.Cursor.IsSet()
}

func validatePagination(pagination CursorPagination) error {
	if pagination.HasForward() && pagination.HasBackward() {
		return errors.New("cannot provide both forward and backward cursors")
	}

	var cursorOffset CursorOffset

	if pagination.HasForward() {
		cursorOffset = *pagination.Forward
	} else if pagination.HasBackward() {
		cursorOffset = *pagination.Backward
	} else {
		// no pagination is provided is okay
		return nil
	}

	limit := *cursorOffset.Limit
	if limit <= 0 || limit > maxPageSize {
		return ErrCursorOverflow
	}

	return nil
}

type PageInfo struct {
	HasNextPage     bool
	HasPreviousPage bool
	StartCursor     string
	EndCursor       string
}

func (p PageInfo) ToProto() *v2.PageInfo {
	return &v2.PageInfo{
		HasNextPage:     p.HasNextPage,
		HasPreviousPage: p.HasPreviousPage,
		StartCursor:     p.StartCursor,
		EndCursor:       p.EndCursor,
	}
}
