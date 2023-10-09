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

	"code.vegaprotocol.io/vega/libs/ptr"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type nodeDataResolver VegaResolverRoot

func toNodeSet(obj *proto.NodeSet) *NodeSet {
	ns := &NodeSet{
		Total:    int(obj.Total),
		Demoted:  obj.Demoted,
		Promoted: obj.Promoted,
		Inactive: int(obj.Inactive),
	}
	if obj.Maximum != nil {
		ns.Maximum = ptr.From(int(*obj.Maximum))
	}
	return ns
}

func (r *nodeDataResolver) TotalNodes(ctx context.Context, obj *proto.NodeData) (int, error) {
	return int(obj.TotalNodes), nil
}

func (r *nodeDataResolver) InactiveNodes(ctx context.Context, obj *proto.NodeData) (int, error) {
	return int(obj.InactiveNodes), nil
}

func (r *nodeDataResolver) Uptime(ctx context.Context, obj *proto.NodeData) (float64, error) {
	return float64(obj.Uptime), nil
}

func (r *nodeDataResolver) TendermintNodes(ctx context.Context, obj *proto.NodeData) (*NodeSet, error) {
	return toNodeSet(obj.TendermintNodes), nil
}

func (r *nodeDataResolver) ErsatzNodes(ctx context.Context, obj *proto.NodeData) (*NodeSet, error) {
	if obj.ErsatzNodes == nil || obj.ErsatzNodes.Total == 0 {
		return nil, nil
	}
	return toNodeSet(obj.ErsatzNodes), nil
}

func (r *nodeDataResolver) PendingNodes(ctx context.Context, obj *proto.NodeData) (*NodeSet, error) {
	if obj.PendingNodes == nil || obj.PendingNodes.Total == 0 {
		return nil, nil
	}
	return toNodeSet(obj.PendingNodes), nil
}
