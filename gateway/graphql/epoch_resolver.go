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
	"strconv"

	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	proto "code.vegaprotocol.io/protos/vega"
)

type epochResolver VegaResolverRoot

func (r *epochResolver) ID(ctx context.Context, obj *proto.Epoch) (string, error) {
	id := strconv.FormatUint(obj.Seq, 10)

	return id, nil
}

func (r *epochResolver) Delegations(
	ctx context.Context,
	obj *proto.Epoch,
	partyID *string,
	nodeID *string,
	skip, first, last *int,
) ([]*proto.Delegation, error) {

	req := &protoapi.DelegationsRequest{
		Pagination: makePagination(skip, first, last),
	}

	if partyID != nil && *partyID != "" {
		req.Party = *partyID
	}

	if nodeID != nil && *nodeID != "" {
		req.NodeId = *nodeID
	}

	resp, err := r.tradingDataClient.Delegations(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Delegations, nil
}
