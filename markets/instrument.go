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

package markets

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/products"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/types"
)

// TradableInstrument represent an instrument to be trade in a market.
type TradableInstrument struct {
	Instrument       *Instrument
	MarginCalculator *types.MarginCalculator
	RiskModel        risk.Model
}

// NewTradableInstrument will instantiate a new tradable instrument
// using a market framework configuration for a tradable instrument.
func NewTradableInstrument(ctx context.Context, log *logging.Logger, pti *types.TradableInstrument, oe products.OracleEngine) (*TradableInstrument, error) {
	instrument, err := NewInstrument(ctx, log, pti.Instrument, oe)
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

func (i *TradableInstrument) UpdateInstrument(ctx context.Context, log *logging.Logger, ti *types.TradableInstrument, oe products.OracleEngine) error {
	i.Instrument.Unsubscribe(ctx)

	instrument, err := NewInstrument(ctx, log, ti.Instrument, oe)
	if err != nil {
		return err
	}

	asset := instrument.Product.GetAsset()

	riskModel, err := risk.NewModel(ti.RiskModel, asset)
	if err != nil {
		return fmt.Errorf("unable to instantiate risk model: %w", err)
	}

	i.Instrument = instrument
	i.RiskModel = riskModel
	i.MarginCalculator = ti.MarginCalculator
	return nil
}

// Instrument represent an instrument used in a market.
type Instrument struct {
	ID       string
	Code     string
	Name     string
	Metadata *types.InstrumentMetadata
	Product  products.Product

	Quote string
}

// NewInstrument will instantiate a new instrument
// using a market framework configuration for a instrument.
func NewInstrument(ctx context.Context, log *logging.Logger, pi *types.Instrument, oe products.OracleEngine) (*Instrument, error) {
	product, err := products.New(ctx, log, pi.Product, oe)
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

func (i *Instrument) Unsubscribe(ctx context.Context) {
	i.Product.Unsubscribe(ctx)
}
