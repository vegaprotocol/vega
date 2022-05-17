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

type CursorBuilder struct {
	ColumnName string
	Sort       Sorting
	Cmp        Compare
	Value      any
}

func NewCursorBuilder(columnName string, sort Sorting, cmp Compare, value any) CursorBuilder {
	return CursorBuilder{
		ColumnName: columnName,
		Sort:       sort,
		Cmp:        cmp,
		Value:      value,
	}
}

func (c CursorBuilder) Where(args ...interface{}) (string, []interface{}) {
	if c.Cmp == "" || c.Value == nil {
		return "", args
	}

	where := fmt.Sprintf("%s %s %v", c.ColumnName, c.Cmp, nextBindVar(&args, c.Value))
	return where, args
}

func (c CursorBuilder) OrderBy() string {
	return fmt.Sprintf("%s %s", c.ColumnName, c.Sort)
}

type CursorBuilders []CursorBuilder

func (c CursorBuilders) Where(args ...interface{}) (string, []interface{}) {
	var where string

	for i, cursor := range c {
		if i > 0 && strings.TrimSpace(where) != "" {
			where = fmt.Sprintf("%s AND", where)
		}
		var cursorCondition string
		cursorCondition, args = cursor.Where(args...)
		where = fmt.Sprintf("%s %s", where, cursorCondition)
	}

	return strings.TrimSpace(where), args
}

func (c CursorBuilders) OrderBy() string {
	var orderBy string

	for i, cursor := range c {
		if i > 0 {
			orderBy = fmt.Sprintf("%s,", orderBy)
		}
		orderBy = fmt.Sprintf("%s %s", orderBy, cursor.OrderBy())
	}

	return strings.TrimSpace(orderBy)
}
