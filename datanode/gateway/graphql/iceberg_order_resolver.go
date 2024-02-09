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

	"code.vegaprotocol.io/vega/protos/vega"
)

type icebergOrderResolver VegaResolverRoot

func (r *icebergOrderResolver) PeakSize(_ context.Context, io *vega.IcebergOrder) (string, error) {
	return strconv.FormatUint(io.PeakSize, 10), nil
}

func (r *icebergOrderResolver) MinimumVisibleSize(_ context.Context, io *vega.IcebergOrder) (string, error) {
	return strconv.FormatUint(io.MinimumVisibleSize, 10), nil
}

func (r *icebergOrderResolver) ReservedRemaining(_ context.Context, io *vega.IcebergOrder) (string, error) {
	return strconv.FormatUint(io.ReservedRemaining, 10), nil
}
