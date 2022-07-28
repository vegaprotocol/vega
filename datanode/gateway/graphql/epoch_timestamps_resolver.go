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
	"context"
	"time"

	"code.vegaprotocol.io/data-node/datanode/vegatime"
	proto "code.vegaprotocol.io/protos/vega"
)

type epochTimestampsResolver VegaResolverRoot

func (r *epochTimestampsResolver) Start(ctx context.Context, obj *proto.EpochTimestamps) (*string, error) {
	var t string
	if obj.StartTime > 0 {
		t = vegatime.UnixNano(obj.StartTime).Format(time.RFC3339)
	}
	return &t, nil
}

func (r *epochTimestampsResolver) End(ctx context.Context, obj *proto.EpochTimestamps) (*string, error) {
	var t string
	if obj.EndTime > 0 {
		t = vegatime.UnixNano(obj.EndTime).Format(time.RFC3339)
	}
	return &t, nil
}

func (r *epochTimestampsResolver) Expiry(ctx context.Context, obj *proto.EpochTimestamps) (*string, error) {
	var t string
	if obj.ExpiryTime > 0 {
		t = vegatime.UnixNano(obj.ExpiryTime).Format(time.RFC3339)
	}
	return &t, nil
}
