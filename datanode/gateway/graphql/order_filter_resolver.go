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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type orderFilterResolver VegaResolverRoot

func (o orderFilterResolver) Status(ctx context.Context, obj *v2.OrderFilter, data []vega.Order_Status) error {
	obj.Statuses = data
	return nil
}

func (o orderFilterResolver) TimeInForce(ctx context.Context, obj *v2.OrderFilter, data []vega.Order_TimeInForce) error {
	obj.TimeInForces = data
	return nil
}
