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

package markets

import (
	"context"
	"fmt"
	"time"

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
	oe products.OracleEngine,
	broker products.Broker,
	productState *snapshotpb.Product,
	assetDP uint32,
	tm time.Time,
) (*Instrument, error) {
	product, err := products.NewFromSnapshot(ctx, log, pi.Product, oe, broker, productState, assetDP, tm)
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
	oe products.OracleEngine,
	broker products.Broker,
	productState *snapshotpb.Product,
	assetDP uint32,
	tm time.Time,
) (*TradableInstrument, error) {
	instrument, err := NewInstrumentFromSnapshot(ctx, log, pti.Instrument, oe, broker, productState, assetDP, tm)
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
