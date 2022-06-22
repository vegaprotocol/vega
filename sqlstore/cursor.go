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

package sqlstore

import (
	"fmt"
	"strings"
)

type (
	Sorting = string
	Compare = string
)

const (
	ASC  Sorting = "ASC"
	DESC Sorting = "DESC"

	EQ Compare = "="
	NE Compare = "!="
	GT Compare = ">"
	LT Compare = "<"
	GE Compare = ">="
	LE Compare = "<="
)

type CursorQueryParameter struct {
	ColumnName string
	Sort       Sorting
	Cmp        Compare
	Value      any
}

func NewCursorQueryParameter(columnName string, sort Sorting, cmp Compare, value any) CursorQueryParameter {
	return CursorQueryParameter{
		ColumnName: columnName,
		Sort:       sort,
		Cmp:        cmp,
		Value:      value,
	}
}

func (c CursorQueryParameter) Where(args ...interface{}) (string, []interface{}) {
	if c.Cmp == "" || c.Value == nil {
		return "", args
	}

	where := fmt.Sprintf("%s %s %v", c.ColumnName, c.Cmp, nextBindVar(&args, c.Value))
	return where, args
}

func (c CursorQueryParameter) OrderBy() string {
	return fmt.Sprintf("%s %s", c.ColumnName, c.Sort)
}

type CursorQueryParameters []CursorQueryParameter

func (c CursorQueryParameters) Where(args ...interface{}) (string, []interface{}) {
	var where string

	for i, cursor := range c {
		var cursorCondition string
		cursorCondition, args = cursor.Where(args...)
		if i > 0 && strings.TrimSpace(where) != "" && strings.TrimSpace(cursorCondition) != "" {
			where = fmt.Sprintf("%s AND", where)
		}
		where = fmt.Sprintf("%s %s", where, cursorCondition)
	}

	return strings.TrimSpace(where), args
}

func (c CursorQueryParameters) OrderBy() string {
	var orderBy string

	for i, cursor := range c {
		if i > 0 {
			orderBy = fmt.Sprintf("%s,", orderBy)
		}
		orderBy = fmt.Sprintf("%s %s", orderBy, cursor.OrderBy())
	}

	return strings.TrimSpace(orderBy)
}
