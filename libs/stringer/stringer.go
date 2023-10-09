// Copyright (C) 2023  Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
