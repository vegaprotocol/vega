package entities

import (
	"encoding/base64"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/pkg/errors"
)

type OffsetPagination struct {
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

type Pagination struct {
	Forward  *offset
	Backward *offset
}

func (p Pagination) HasForward() bool {
	return p.Forward != nil
}

func (p Pagination) HasBackward() bool {
	return p.Backward != nil
}

func PaginationFromProto(cp *v2.Pagination) (Pagination, error) {
	if cp == nil {
		return Pagination{}, nil
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
				return Pagination{}, errors.Wrap(err, "failed to decode after cursor")
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
				return Pagination{}, errors.Wrap(err, "failed to decode before cursor")
			}
			backwardOffset.Cursor = &before
		}
	}

	pagination := Pagination{
		Forward:  forwardOffset,
		Backward: backwardOffset,
	}

	if err = validatePagination(pagination); err != nil {
		return Pagination{}, err
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

func validatePagination(pagination Pagination) error {
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
