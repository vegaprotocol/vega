// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package products

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var ErrNoStateProvidedForPerpsWithSnapshot = errors.New("no state provided to restore perps from a snapshot")

// New instance a new product from a Market framework product configuration.
func NewFromSnapshot(
	ctx context.Context,
	log *logging.Logger,
	pp interface{},
	oe OracleEngine,
	broker Broker,
	state *snapshotpb.Product,
) (Product, error) {
	if pp == nil {
		return nil, ErrNilProduct
	}

	switch p := pp.(type) {
	case *types.InstrumentFuture: // no state in the future, so all OK
		return NewFuture(ctx, log, p.Future, oe)
	case *types.InstrumentPerpetual:
		perpsState := state.GetPerps()
		if perpsState == nil {
			return nil, ErrNoStateProvidedForPerpsWithSnapshot
		}
		return NewPerpetualFromSnapshot(ctx, log, p.Perpetual, oe, broker, perpsState)
	default:
		return nil, ErrUnimplementedProduct
	}
}
