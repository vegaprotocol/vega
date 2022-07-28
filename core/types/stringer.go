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

package types

import (
	"reflect"
	"strconv"

	"code.vegaprotocol.io/vega/core/types/num"
)

type Stringer interface {
	String() string
}

func reflectPointerToString(obj Stringer) string {
	if obj == nil || reflect.ValueOf(obj).Kind() == reflect.Ptr && reflect.ValueOf(obj).IsNil() {
		return "nil"
	}
	return obj.String()
}

func uintPointerToString(obj *num.Uint) string {
	if obj == nil {
		return "nil"
	}
	return obj.String()
}

func int64PointerToString(n *int64) string {
	if n == nil {
		return "nil"
	}
	return strconv.FormatInt(*n, 10)
}
