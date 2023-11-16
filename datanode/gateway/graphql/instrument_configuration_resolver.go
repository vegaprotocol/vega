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
	"errors"

	types "code.vegaprotocol.io/vega/protos/vega"
)

type myInstrumentConfigurationResolver VegaResolverRoot

func (r *myInstrumentConfigurationResolver) FutureProduct(ctx context.Context, obj *types.InstrumentConfiguration) (*types.FutureProduct, error) {
	return obj.GetFuture(), nil
}

func (r *myInstrumentConfigurationResolver) Product(ctx context.Context, obj *types.InstrumentConfiguration) (ProductConfiguration, error) {
	switch obj.GetProduct().(type) {
	case *types.InstrumentConfiguration_Future:
		return obj.GetFuture(), nil
	case *types.InstrumentConfiguration_Spot:
		return obj.GetSpot(), nil
	case *types.InstrumentConfiguration_Perpetual:
		return obj.GetPerpetual(), nil
	default:
		return nil, errors.New("unknown product type")
	}
}
