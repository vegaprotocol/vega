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
