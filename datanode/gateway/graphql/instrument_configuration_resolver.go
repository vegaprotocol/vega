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
