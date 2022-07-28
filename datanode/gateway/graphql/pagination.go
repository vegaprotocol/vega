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

package gql

import (
	"code.vegaprotocol.io/protos/commands"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
)

var defaultPagination = protoapi.Pagination{
	Skip:       0,
	Limit:      50,
	Descending: true,
}

func (p *OffsetPagination) ToProto() (protoapi.Pagination, error) {
	if p == nil {
		return defaultPagination, nil
	}

	if p.Skip < 0 {
		return protoapi.Pagination{}, commands.ErrMustBePositiveOrZero
	}

	if p.Limit < 0 {
		return protoapi.Pagination{}, commands.ErrMustBePositiveOrZero
	}

	return protoapi.Pagination{
		Skip:       uint64(p.Skip),
		Limit:      uint64(p.Limit),
		Descending: p.Descending,
	}, nil
}
