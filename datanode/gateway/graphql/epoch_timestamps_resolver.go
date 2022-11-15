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

	"code.vegaprotocol.io/vega/libs/ptr"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type epochTimestampsResolver VegaResolverRoot

func (r *epochTimestampsResolver) Start(ctx context.Context, obj *proto.EpochTimestamps) (*int64, error) {
	var t *int64
	if obj.StartTime > 0 {
		t = ptr.From(obj.StartTime)
	}
	return t, nil
}

func (r *epochTimestampsResolver) End(ctx context.Context, obj *proto.EpochTimestamps) (*int64, error) {
	var t *int64
	if obj.EndTime > 0 {
		t = ptr.From(obj.EndTime)
	}
	return t, nil
}

func (r *epochTimestampsResolver) Expiry(ctx context.Context, obj *proto.EpochTimestamps) (*int64, error) {
	var t *int64
	if obj.ExpiryTime > 0 {
		t = ptr.From(obj.ExpiryTime)
	}
	return t, nil
}

func (r *epochTimestampsResolver) FirstBlock(_ context.Context, obj *proto.EpochTimestamps) (string, error) {
	return strconv.FormatUint(obj.FirstBlock, 10), nil
}

func (r *epochTimestampsResolver) LastBlock(_ context.Context, obj *proto.EpochTimestamps) (*string, error) {
	var ret *string
	if obj.LastBlock > 0 {
		lastBlock := strconv.FormatUint(obj.LastBlock, 10)
		ret = &lastBlock
	}
	return ret, nil
}
