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

func orderAndPaginateWithCursor(query string, cursor entities.Pagination, cursorColumn string,
	args ...interface{}) (string, []interface{},
) {
	var limit int32
	var order string

	whereOrAnd := "WHERE"

	if strings.Contains(strings.ToUpper(query), "WHERE") {
		whereOrAnd = "AND"
	}

	if cursor.HasForward() && cursor.Forward.Limit != nil {
		order = "ASC"
		limit = *cursor.Forward.Limit + 1
		if cursor.Forward.HasCursor() {
			query = fmt.Sprintf("%s %s %s >= %s", query, whereOrAnd, cursorColumn, nextBindVar(&args, cursor.Forward.Cursor.Value()))
			limit = *cursor.Forward.Limit + 2 // +2 to make sure we get the previous and next cursor
		}
	} else if cursor.HasBackward() && cursor.Backward.Limit != nil {
		order = "DESC"
		limit = *cursor.Backward.Limit + 1
		if cursor.Backward.HasCursor() {
			query = fmt.Sprintf("%s %s %s <= %s", query, whereOrAnd, cursorColumn, nextBindVar(&args, cursor.Backward.Cursor.Value()))
			limit = *cursor.Backward.Limit + 2 // +2 to make sure we get the previous and next cursor
		}
	} else {
		// return everything ordered by the cursor column ordered ascending
		order = "ASC"
		query = fmt.Sprintf("%s ORDER BY %s %s", query, cursorColumn, order)
		return query, args
	}

	query = fmt.Sprintf("%s ORDER BY %s %s", query, cursorColumn, order)
	query = fmt.Sprintf("%s LIMIT %d", query, limit)

	return query, args
}
