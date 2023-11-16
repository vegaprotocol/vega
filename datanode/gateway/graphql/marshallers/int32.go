// Copyright (C) 2023 Gobalsky Labs Limited
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

package marshallers

import (
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

func MarshalUint32(t uint32) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, fmt.Sprint(t))
	})
}

func UnmarshalUint32(v interface{}) (uint32, error) {
	s, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("Expected uint32 to be a string")
	}

	value64, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("Couldn't parse value into uint32")
	}

	return uint32(value64), nil
}
