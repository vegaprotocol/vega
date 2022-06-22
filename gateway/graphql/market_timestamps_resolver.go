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

	"code.vegaprotocol.io/data-node/vegatime"
	proto "code.vegaprotocol.io/protos/vega"
)

type marketTimestampsResolver VegaResolverRoot

func (r *marketTimestampsResolver) Proposed(ctx context.Context, obj *proto.MarketTimestamps) (*string, error) {
	if obj.Proposed == 0 {
		return nil, nil
	}
	value := vegatime.Format(vegatime.UnixNano(obj.Proposed))
	return &value, nil
}

func (r *marketTimestampsResolver) Pending(ctx context.Context, obj *proto.MarketTimestamps) (*string, error) {
	if obj.Pending == 0 {
		return nil, nil
	}
	value := vegatime.Format(vegatime.UnixNano(obj.Pending))
	return &value, nil
}

func (r *marketTimestampsResolver) Open(ctx context.Context, obj *proto.MarketTimestamps) (*string, error) {
	if obj.Open == 0 {
		return nil, nil
	}
	value := vegatime.Format(vegatime.UnixNano(obj.Open))
	return &value, nil
}

func (r *marketTimestampsResolver) Close(ctx context.Context, obj *proto.MarketTimestamps) (*string, error) {
	if obj.Close == 0 {
		return nil, nil
	}
	value := vegatime.Format(vegatime.UnixNano(obj.Close))
	return &value, nil
}
