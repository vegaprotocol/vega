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
	"strconv"
	"strings"

	"code.vegaprotocol.io/data-node/entities"
)

// A handy little helper function for building queries. Appends 'value'
// to the 'args' slice and returns a string '$N' referring to the index
// of the value in args. For example:
//
// 	var args []interface{}
//  query = "select * from foo where id=" + nextBindVar(&args, 100)
//  db.Query(query, args...)
func nextBindVar(args *[]interface{}, value interface{}) string {
	*args = append(*args, value)
	return "$" + strconv.Itoa(len(*args))
}

// orderAndPaginateQuery is a helper function to simplify adding ordering and pagination statements to the end of a query
// with the appropriate binding variables amd returns the query string and list of arguments to pass to the query execution handler
func orderAndPaginateQuery(query string, orderColumns []string, pagination entities.OffsetPagination, args ...interface{}) (string, []interface{}) {
	ordering := "ASC"

	if pagination.Descending {
		ordering = "DESC"
	}

	sbOrderBy := strings.Builder{}

	if len(orderColumns) > 0 {
		sbOrderBy.WriteString("ORDER BY")

		sep := ""

		for _, column := range orderColumns {
			sbOrderBy.WriteString(fmt.Sprintf("%s %s %s", sep, column, ordering))
			sep = ","
		}
	}

	var paging string

	if pagination.Skip != 0 {
		paging = fmt.Sprintf("%sOFFSET %s ", paging, nextBindVar(&args, pagination.Skip))
	}

	if pagination.Limit != 0 {
		paging = fmt.Sprintf("%sLIMIT %s ", paging, nextBindVar(&args, pagination.Limit))
	}

	query = fmt.Sprintf("%s %s %s", query, sbOrderBy.String(), paging)

	return query, args
}

func orderAndPaginateWithCursor(query string, pagination entities.CursorPagination, cursors CursorQueryParameters,
	args ...interface{}) (string, []interface{},
) {
	var order string

	whereOrAnd := "WHERE"

	if strings.Contains(strings.ToUpper(query), "WHERE") {
		whereOrAnd = "AND"
	}

	var cursor string
	cursor, args = cursors.Where(args...)
	if cursor != "" {
		query = fmt.Sprintf("%s %s %s", query, whereOrAnd, cursor)
	}

	limit := calculateLimit(pagination)

	if limit == 0 {
		// return everything ordered by the cursor column ordered ascending
		order = cursors.OrderBy()
		query = fmt.Sprintf("%s ORDER BY %s", query, order)
		return query, args
	}

	order = cursors.OrderBy()
	query = fmt.Sprintf("%s ORDER BY %s", query, order)
	query = fmt.Sprintf("%s LIMIT %d", query, limit)

	return query, args
}

func calculateLimit(pagination entities.CursorPagination) int {
	var limit int32
	if pagination.HasForward() && pagination.Forward.Limit != nil {
		limit = *pagination.Forward.Limit + 1
		if pagination.Forward.HasCursor() {
			limit = *pagination.Forward.Limit + 2 // +2 to make sure we get the previous and next cursor
		}
	} else if pagination.HasBackward() && pagination.Backward.Limit != nil {
		limit = *pagination.Backward.Limit + 1
		if pagination.Backward.HasCursor() {
			limit = *pagination.Backward.Limit + 2 // +2 to make sure we get the previous and next cursor
		}
	}

	return int(limit)
}

func extractPaginationInfo(pagination entities.CursorPagination) (Sorting, Compare, string) {
	var cmp Compare
	var value string

	sort := ASC

	if pagination.NewestFirst {
		sort = DESC
	}

	if pagination.HasForward() {
		if pagination.Forward.HasCursor() {
			cmp = GE
			if pagination.NewestFirst {
				cmp = LE
			}
			value = pagination.Forward.Cursor.Value()
		}
	} else if pagination.HasBackward() {
		sort = DESC

		if pagination.NewestFirst {
			sort = ASC
		}

		if pagination.Backward.HasCursor() {
			cmp = LE
			if pagination.NewestFirst {
				cmp = GE
			}
			value = pagination.Backward.Cursor.Value()
		}
	}

	return sort, cmp, value
}
