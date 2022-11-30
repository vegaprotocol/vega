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
	"errors"
	"io"
	"strconv"

	"code.vegaprotocol.io/vega/datanode/vegatime"
	"github.com/99designs/gqlgen/graphql"
)

func MarshalTimestamp(t int64) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		// special case for 0 timestamps, returns null
		if t == 0 {
			io.WriteString(w, "null")
			return
		}
		io.WriteString(w, strconv.Quote(vegatime.Format(vegatime.UnixNano(t))))
	})
}

func UnmarshalTimestamp(v interface{}) (int64, error) {
	s, ok := v.(string)
	if !ok {
		return 0, errors.New("expected timestamp to be a string")
	}

	t, err := vegatime.Parse(s)
	if err != nil {
		return 0, err
	}

	return t.UnixNano(), nil
}
