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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type epochResolver VegaResolverRoot

func (r *epochResolver) ID(_ context.Context, obj *proto.Epoch) (string, error) {
	return strconv.FormatUint(obj.Seq, 10), nil
}

func (r *epochResolver) ValidatorsConnection(ctx context.Context, epoch *proto.Epoch, pagination *v2.Pagination) (*v2.NodesConnection, error) {
	var epochSeq *uint64
	if epoch != nil {
		epochSeq = &epoch.Seq
	}

	resp, err := r.tradingDataClientV2.ListNodes(ctx, &v2.ListNodesRequest{
		EpochSeq:   epochSeq,
		Pagination: pagination,
	})
	if err != nil {
		return nil, err
	}

	return resp.Nodes, err
}

func (r *epochResolver) DelegationsConnection(ctx context.Context, epoch *proto.Epoch, partyID *string, nodeID *string, pagination *v2.Pagination) (*v2.DelegationsConnection, error) {
	var epochID *string

	if epoch != nil {
		seq := strconv.FormatUint(epoch.Seq, 10)
		epochID = &seq
	}

	return handleDelegationConnectionRequest(ctx, r.tradingDataClientV2, partyID, nodeID, epochID, pagination)
}
