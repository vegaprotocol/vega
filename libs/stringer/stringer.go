// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package stringer

import (
	"reflect"
	"strconv"

	"code.vegaprotocol.io/vega/libs/num"
)

type Stringer interface {
	String() string
}

func ReflectPointerToString(obj Stringer) string {
	if obj == nil || reflect.ValueOf(obj).Kind() == reflect.Ptr && reflect.ValueOf(obj).IsNil() {
		return "nil"
	}
	return obj.String()
}

func UintPointerToString(obj *num.Uint) string {
	if obj == nil {
		return "nil"
	}
	return obj.String()
}

func Int64PointerToString(n *int64) string {
	if n == nil {
		return "nil"
	}
	return strconv.FormatInt(*n, 10)
}
