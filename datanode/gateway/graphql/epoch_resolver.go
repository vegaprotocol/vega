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

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/data-node/api/v1"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type epochResolver VegaResolverRoot

func (r *epochResolver) ID(_ context.Context, obj *proto.Epoch) (string, error) {
	id := strconv.FormatUint(obj.Seq, 10)

	return id, nil
}

// Deprecated: Use DelegationsConnection instead.
func (r *epochResolver) Delegations(
	ctx context.Context,
	_ *proto.Epoch,
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
	header := metadata.MD{}
	resp, err := r.tradingDataClient.Delegations(ctx, req, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Delegations, nil
}

func (r *epochResolver) ValidatorsConnection(ctx context.Context, epoch *proto.Epoch, pagination *v2.Pagination) (*v2.NodesConnection, error) {
	var epochSeq *uint64
	if epoch != nil {
		epochSeq = &epoch.Seq
	}
	header := metadata.MD{}
	resp, err := r.tradingDataClientV2.ListNodes(ctx, &v2.ListNodesRequest{
		EpochSeq:   epochSeq,
		Pagination: pagination,
	}, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		r.log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Nodes, nil
}

func (r *epochResolver) DelegationsConnection(ctx context.Context, epoch *proto.Epoch, partyID *string, nodeID *string, pagination *v2.Pagination) (*v2.DelegationsConnection, error) {
	var epochID *string

	if epoch != nil {
		seq := strconv.FormatUint(epoch.Seq, 10)
		epochID = &seq
	}

	return handleDelegationConnectionRequest(ctx, r.tradingDataClientV2, partyID, nodeID, epochID, pagination, r.log)
}
