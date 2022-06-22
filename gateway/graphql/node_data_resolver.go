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

	proto "code.vegaprotocol.io/protos/vega"
)

type nodeDataResolver VegaResolverRoot

func (r *nodeDataResolver) TotalNodes(ctx context.Context, obj *proto.NodeData) (int, error) {
	return int(obj.TotalNodes), nil
}

func (r *nodeDataResolver) InactiveNodes(ctx context.Context, obj *proto.NodeData) (int, error) {
	return int(obj.InactiveNodes), nil
}

func (r *nodeDataResolver) ValidatingNodes(ctx context.Context, obj *proto.NodeData) (int, error) {
	return int(obj.ValidatingNodes), nil
}

func (r *nodeDataResolver) Uptime(ctx context.Context, obj *proto.NodeData) (float64, error) {
	return float64(obj.Uptime), nil
}
