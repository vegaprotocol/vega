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

package sqlstore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
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

type ColumnOrdering struct {
	// Name of the column in the database table to match to the struct field
	Name string
	// Sorting is the sorting order to use for the column
	Sorting Sorting
	// Prefix is the prefix to add to the column name in order to resolve duplicate
	// column names that might be in the query
	Prefix string
}

func NewColumnOrdering(name string, sorting Sorting) ColumnOrdering {
	return ColumnOrdering{Name: name, Sorting: sorting}
}

type TableOrdering []ColumnOrdering

func (t *TableOrdering) OrderByClause() string {
	if len(*t) == 0 {
		return ""
	}

	fragments := make([]string, len(*t))
	for i, column := range *t {
		prefix := column.Prefix
		if column.Prefix != "" && !strings.HasSuffix(column.Prefix, ".") {
			prefix += "."
		}
		fragments[i] = fmt.Sprintf("%s%s %s", prefix, column.Name, column.Sorting)
	}
	return fmt.Sprintf("ORDER BY %s", strings.Join(fragments, ","))
}

func (t *TableOrdering) Reversed() TableOrdering {
	reversed := make([]ColumnOrdering, len(*t))
	for i, column := range *t {
		if column.Sorting == DESC {
			reversed[i] = ColumnOrdering{Name: column.Name, Sorting: ASC}
		}
		if column.Sorting == ASC {
			reversed[i] = ColumnOrdering{Name: column.Name, Sorting: DESC}
		}
	}
	return reversed
}

// CursorPredicate generates an SQL predicate which excludes all rows before the supplied cursor,
// with regards to the supplied table ordering. The values used for comparison are added to
// the args list and bind variables used in the query fragment.
//
// For example, with if you had a query with columns sorted foo ASCENDING, bar DESCENDING and a
// cursor with {foo=1, bar=2}, it would yield a string predicate like this:
//
// (foo > $1) OR (foo = $1 AND bar <= $2)
//
// And 'args' would have 1 and 2 appended to it.
//
// Notes:
//   - The predicate *includes* the value at the cursor
//   - Only fields that are present in both the cursor and the ordering are considered
//   - The union of those fields must have enough information to uniquely identify a row
//   - The table ordering must be sufficient to ensure that a row identified by a cursor cannot
//     change position in relation to the other rows
func CursorPredicate(args []interface{}, cursor interface{}, ordering TableOrdering) (string, []interface{}, error) {
	cursorPredicates := []string{}
	equalPredicates := []string{}

	for i, column := range ordering {
		// For the non-last columns, use LT/GT, so we don't include stuff before the cursor
		var operator string
		if column.Sorting == ASC {
			operator = ">"
		} else if column.Sorting == DESC {
			operator = "<"
		} else {
			return "", nil, fmt.Errorf("unknown sort direction %s", column.Sorting)
		}

		// For the last column, we want to use GTE/LTE so we include the value at the cursor
		isLast := i == (len(ordering) - 1)
		if isLast {
			operator = operator + "="
		}

		value, err := StructValueForColumn(cursor, column.Name)
		if err != nil {
			return "", nil, err
		}

		prefix := column.Prefix
		if column.Prefix != "" && !strings.HasSuffix(column.Prefix, ".") {
			prefix += "."
		}

		bindVar := nextBindVar(&args, value)
		inequalityPredicate := fmt.Sprintf("%s%s %s %s", prefix, column.Name, operator, bindVar)

		colPredicates := append(equalPredicates, inequalityPredicate)
		colPredicateString := strings.Join(colPredicates, " AND ")
		colPredicateString = fmt.Sprintf("(%s)", colPredicateString)
		cursorPredicates = append(cursorPredicates, colPredicateString)

		equalityPredicate := fmt.Sprintf("%s%s = %s", prefix, column.Name, bindVar)
		equalPredicates = append(equalPredicates, equalityPredicate)
	}

	predicateString := strings.Join(cursorPredicates, " OR ")

	return predicateString, args, nil
}

type parser interface {
	Parse(string) error
}

// This is a bit magical, it allows us to use the real cursor type for instantiation and the pointer
// type for calling methods with pointer receivers (e.g. Parse) for details see
// https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#pointer-method-example
type parserPtr[T any] interface {
	parser
	*T
}

// We have to roll our own equals function here for comparing the cursors because some cursor parameters use
// types that do not implement `comparable`.
func equals[T any](actual, other T) (bool, error) {
	var a, b bytes.Buffer
	enc := gob.NewEncoder(&a)
	err := enc.Encode(actual)
	if err != nil {
		return false, err
	}

	enc = gob.NewEncoder(&b)
	err = enc.Encode(other)
	if err != nil {
		return false, err
	}

	return bytes.Equal(a.Bytes(), b.Bytes()), nil
}

// PaginateQuery takes a query string & bind arg list and returns the same with additional SQL to
//   - exclude rows before the cursor (or after it if the cursor is a backwards looking one)
//   - limit the number of rows to the pagination limit +1 (no cursor) or +2 (cursor)
//     [for purposes of later figuring out whether there are next or previous pages]
//   - order the query according to the TableOrdering supplied
//     the order is reversed if pagination request is backwards
//
// For example with cursor to a row where foo=42, and a pagination saying get the next 3 then:
// PaginateQuery[MyCursor]("SELECT foo FROM my_table", args, ordering, pagination)
//
// Would append `42` to the arg list and return
// SELECT foo FROM my_table WHERE foo>=$1 ORDER BY foo ASC LIMIT 5
//
// See CursorPredicate() for more details about how the cursor filtering is done.
func PaginateQuery[T any, PT parserPtr[T]](
	query string,
	args []interface{},
	ordering TableOrdering,
	pagination entities.CursorPagination,
) (string, []interface{}, error) {
	return paginateQueryInternal[T, PT](query, args, ordering, pagination, false)
}

func PaginateQueryWithoutOrderBy[T any, PT parserPtr[T]](
	query string,
	args []interface{},
	ordering TableOrdering,
	pagination entities.CursorPagination,
) (string, []interface{}, error) {
	return paginateQueryInternal[T, PT](query, args, ordering, pagination, true)
}

func paginateQueryInternal[T any, PT parserPtr[T]](
	query string,
	args []interface{},
	ordering TableOrdering,
	pagination entities.CursorPagination,
	omitOrderBy bool,
) (string, []interface{}, error) {
	// Extract a cursor struct from the pagination struct
	cursor, err := parseCursor[T, PT](pagination)
	if err != nil {
		return "", nil, fmt.Errorf("parsing cursor: %w", err)
	}

	// If we're fetching rows before the cursor, reverse the ordering
	if (pagination.HasBackward() && !pagination.NewestFirst) || // Navigating backwards in time order
		(pagination.HasForward() && pagination.NewestFirst) || // Navigating forward in reverse time order
		(!pagination.HasBackward() && !pagination.HasForward() && pagination.NewestFirst) { // No pagination provided, but in reverse time order
		ordering = ordering.Reversed()
	}

	// If the cursor wasn't empty, exclude rows preceding the cursor's row
	var emptyCursor T
	isEmpty, err := equals[T](cursor, emptyCursor)
	if err != nil {
		return "", nil, fmt.Errorf("checking empty cursor: %w", err)
	}
	if !isEmpty {
		whereOrAnd := "WHERE"
		if strings.Contains(strings.ToUpper(query), "WHERE") {
			whereOrAnd = "AND"
		}

		var predicate string
		predicate, args, err = CursorPredicate(args, cursor, ordering)
		if err != nil {
			return "", nil, fmt.Errorf("building cursor predicate: %w", err)
		}
		query = fmt.Sprintf("%s %s (%s)", query, whereOrAnd, predicate)
	}

	// Add an ORDER BY clause if requested
	if !omitOrderBy {
		query = fmt.Sprintf("%s %s", query, ordering.OrderByClause())
	}

	// And a LIMIT clause
	limit := calculateLimit(pagination)
	if limit != 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	return query, args, nil
}

func parseCursor[T any, PT parserPtr[T]](pagination entities.CursorPagination) (T, error) {
	cursor := PT(new(T))

	cursorStr := ""
	if pagination.HasForward() && pagination.Forward.HasCursor() {
		cursorStr = pagination.Forward.Cursor.Value()
	} else if pagination.HasBackward() && pagination.Backward.HasCursor() {
		cursorStr = pagination.Backward.Cursor.Value()
	}

	if cursorStr != "" {
		err := cursor.Parse(cursorStr)
		if err != nil {
			return *cursor, fmt.Errorf("parsing cursor: %w", err)
		}
	}
	return *cursor, nil
}

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
