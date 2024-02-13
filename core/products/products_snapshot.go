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
	marketID string,
	ts TimeService,
	oe OracleEngine,
	broker Broker,
	state *snapshotpb.Product,
	assetDP uint32,
) (Product, error) {
	if pp == nil {
		return nil, ErrNilProduct
	}

	switch p := pp.(type) {
	case *types.InstrumentFuture: // no state in the future, so all OK
		return NewFuture(ctx, log, p.Future, oe, assetDP)
	case *types.InstrumentPerps:
		perpsState := state.GetPerps()
		if perpsState == nil {
			return nil, ErrNoStateProvidedForPerpsWithSnapshot
		}
		return NewPerpetualFromSnapshot(ctx, log, p.Perps, marketID, ts, oe, broker, perpsState, assetDP)
	default:
		return nil, ErrUnimplementedProduct
	}
}
