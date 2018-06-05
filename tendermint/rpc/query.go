package rpc

import (
	"errors"
	"fmt"
	"strings"
)

// Errors relating to Query values.
var (
	ErrMissingFilterConditions = errors.New("rpc: missing filter conditions for Query")
)

// Query represents a query string for certain Tendermint RPC calls. Queries are
// composed of conditions which are of the form:
//
//     <Tag> <Operator> <Value>
//
// e.g.
//
//     tm.event = 'NewBlock'
//
// Tendermint provides a few predefined tags:
//
//     tm.event
//     tx.hash
//     tx.height
//
// App-specific tags can be created on the fly by specifying key/value pairs in
// the DeliverTx.Tags response within the ABCI app, e.g.
//
//     DeliverTx{Tags: []*KVPair{"agent.name": "007"}}
//
// These tags can then be used in queries, e.g.
//
//     agent.name = '007'
//
// Operators can be one of:
//
//     =
//     <
//     <=
//     >
//     >=
//     CONTAINS                  // Only on string values
//
// Values can be strings, numbers, dates, or times, e.g.
//
//     tx.hash = 'DEADBEEF'   // String values must be wrapped in single quotes
//     tx.height = 5
//     contract.expiry_date <= DATE 2018-01-01
//     account.created_at >= TIME 2018-06-04T12:13:00Z
//
//
// And, finally, multiple conditions can be AND-ed together as part of query
// using the form:
//
//     <condition> AND <condition>
//
// e.g.
//
//     tm.event = 'Tx' AND tx.hash = 'DEADBEEF'
type Query struct {
	conditions []string
	err        error
}

// Add lets you add a raw condition string to the Query.
func (q *Query) Add(condition string) *Query {
	if q.err != nil {
		return q
	}
	q.conditions = append(q.conditions, condition)
	return q
}

// Expression returns the query as a string. It is mainly for use by RPC methods
// on the Client, but can also be used for debugging purposes.
func (q *Query) Expression() (string, error) {
	if q.err != nil {
		return "", q.err
	}
	switch len(q.conditions) {
	case 0:
		return "", ErrMissingFilterConditions
	case 1:
		return q.conditions[0], nil
	}
	return strings.Join(q.conditions, " AND "), nil
}

// Filter adds a condition to the underlying Query. It takes a criteria of the
// form:
//
//     <TAG> <OPERATOR>
//
// e.g.
//
//     'contract.ends <'
//     'org.name CONTAINS'
//     'tx.height ='
//
// The tag name can be composed of any ASCII alphanumeric character, hyphen,
// underscore, and period. For the purposes of consistency, a <type>.<member>
// format is encouraged.
//
// The value type is currently limited to just string, int, int64, uint64, and he method can be used like:
//
//     q.Filter("")
//
// Filter mutates the underlying Query and returns it so that it can be used in
// a fluent/chained manner. If any errors are encountered whilst adding the
// filter, then the error is preserved and returned when the Expression method
// is eventually called.
func (q *Query) Filter(criteria string, value interface{}) *Query {
	if q.err != nil {
		return q
	}
	split := strings.Split(criteria, " ")
	if len(split) != 2 {
		q.err = fmt.Errorf(
			"rpc: expected a single space between the query condition's tag and operator: %q",
			criteria)
		return q
	}
	// Limit tag to just ASCII alphanumeric characters, hyphen, underscore, and
	// period for now.
	return q
}
