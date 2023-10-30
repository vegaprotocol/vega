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

package markets

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

// NewInstrument will instantiate a new instrument
// using a market framework configuration for a instrument.
func NewInstrumentFromSnapshot(
	ctx context.Context,
	log *logging.Logger,
	pi *types.Instrument,
	marketID string,
	ts common.TimeService,
	oe products.OracleEngine,
	broker products.Broker,
	productState *snapshotpb.Product,
	assetDP uint32,
) (*Instrument, error) {
	product, err := products.NewFromSnapshot(ctx, log, pi.Product, marketID, ts, oe, broker, productState, assetDP)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate product from instrument configuration: %w", err)
	}
	return &Instrument{
		ID:       pi.ID,
		Code:     pi.Code,
		Name:     pi.Name,
		Metadata: pi.Metadata,
		Product:  product,
	}, err
}

// NewTradableInstrument will instantiate a new tradable instrument
// using a market framework configuration for a tradable instrument.
func NewTradableInstrumentFromSnapshot(
	ctx context.Context,
	log *logging.Logger,
	pti *types.TradableInstrument,
	marketID string,
	ts common.TimeService,
	oe products.OracleEngine,
	broker products.Broker,
	productState *snapshotpb.Product,
	assetDP uint32,
) (*TradableInstrument, error) {
	instrument, err := NewInstrumentFromSnapshot(ctx, log, pti.Instrument, marketID, ts, oe, broker, productState, assetDP)
	if err != nil {
		return nil, err
	}
	asset := instrument.Product.GetAsset()
	riskModel, err := risk.NewModel(pti.RiskModel, asset)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate risk model: %w", err)
	}
	return &TradableInstrument{
		Instrument:       instrument,
		MarginCalculator: pti.MarginCalculator,
		RiskModel:        riskModel,
	}, nil
}
