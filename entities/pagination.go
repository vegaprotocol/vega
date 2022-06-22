// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/pkg/errors"
)

type Pagination interface{}

type OffsetPagination struct {
	Pagination
	Skip       uint64
	Limit      uint64
	Descending bool
}

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

func OffsetPaginationFromProto(pp *v2.OffsetPagination) OffsetPagination {
	return OffsetPagination{
		Skip:       pp.Skip,
		Limit:      pp.Limit,
		Descending: pp.Descending,
	}
}

type CursorPagination struct {
	Pagination
	Forward  *offset
	Backward *offset
}

func (p CursorPagination) HasForward() bool {
	return p.Forward != nil
}

func (p CursorPagination) HasBackward() bool {
	return p.Backward != nil
}

func NewCursorPagination(first *int32, after *string, last *int32, before *string) (CursorPagination, error) {
	return CursorPaginationFromProto(&v2.Pagination{
		First:  first,
		After:  after,
		Last:   last,
		Before: before,
	})
}

func CursorPaginationFromProto(cp *v2.Pagination) (CursorPagination, error) {
	if cp == nil {
		return CursorPagination{}, nil
	}

	var after, before Cursor
	var err error
	var forwardOffset, backwardOffset *offset

	if cp.First != nil {
		forwardOffset = &offset{
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
		backwardOffset = &offset{
			Limit: cp.Last,
		}
		// Proto cursors should be encoded values, so we want to decode them in order to use them
		if cp.Before != nil {
			if err = before.Decode(*cp.Before); err != nil {
				return CursorPagination{}, errors.Wrap(err, "failed to decode before cursor")
			}
			backwardOffset.Cursor = &before
		}
	}

	pagination := CursorPagination{
		Forward:  forwardOffset,
		Backward: backwardOffset,
	}

	if err = validatePagination(pagination); err != nil {
		return CursorPagination{}, err
	}

	return pagination, nil
}

type offset struct {
	Limit  *int32
	Cursor *Cursor
}

func (o offset) IsSet() bool {
	return o.Limit != nil
}

func (o offset) HasCursor() bool {
	return o.Cursor != nil && o.Cursor.IsSet()
}

func validatePagination(pagination CursorPagination) error {
	if pagination.HasForward() && pagination.HasBackward() {
		return errors.New("cannot provide both forward and backward cursors")
	}

	var cursorOffset offset

	if pagination.HasForward() {
		cursorOffset = *pagination.Forward
	} else if pagination.HasBackward() {
		cursorOffset = *pagination.Backward
	} else {
		// no pagination is provided is okay
		return nil
	}

	limit := *cursorOffset.Limit
	if limit <= 0 {
		return errors.New("pagination limit must be greater than 0")
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
