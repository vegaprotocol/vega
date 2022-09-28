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

package store

import "strconv"

// A handy little helper function for building queries. Appends 'value'
// to the 'args' slice and returns a string '$N' referring to the index
// of the value in args. For example:
//
//		var args []interface{}
//	 query = "select * from foo where id=" + nextBindVar(&args, 100)
//	 db.Query(query, args...)
func nextBindVar(args *[]interface{}, value interface{}) string {
	*args = append(*args, value)
	return "$" + strconv.Itoa(len(*args))
}
