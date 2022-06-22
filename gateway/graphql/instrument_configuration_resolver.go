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

	types "code.vegaprotocol.io/protos/vega"
)

type myInstrumentConfigurationResolver VegaResolverRoot

func (r *myInstrumentConfigurationResolver) FutureProduct(ctx context.Context, obj *types.InstrumentConfiguration) (*types.FutureProduct, error) {
	return obj.GetFuture(), nil
}

// func (r *myInstrumentConfigurationResolver) Metadata(ctx context.Context, obj *types.InstrumentConfiguration) (*InstrumentMetadata, error) {
// 	return InstrumentMetadataFromProto(obj.Metadata)
// }
// func (r *myInstrumentResolver) Product(ctx context.Context, obj *proto.Instrument) (Product, error) {
// 	return obj.GetFuture(), nil
//}
