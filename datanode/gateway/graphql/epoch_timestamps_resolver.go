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
