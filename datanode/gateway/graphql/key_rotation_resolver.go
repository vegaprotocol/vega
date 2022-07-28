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

package gql

import (
	"context"
	"strconv"

	v12 "code.vegaprotocol.io/protos/data-node/api/v1"
)

type keyRotationResolver VegaResolverRoot

func (r *keyRotationResolver) BlockHeight(ctx context.Context, obj *v12.KeyRotation) (string, error) {
	return strconv.FormatUint(obj.BlockHeight, 10), nil
}
