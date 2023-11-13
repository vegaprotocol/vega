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
)

// TradableInstrument represent an instrument to be trade in a market.
type TradableInstrument struct {
	Instrument       *Instrument
	MarginCalculator *types.MarginCalculator
	RiskModel        risk.Model
	assetDP          uint32
}

// NewTradableInstrument will instantiate a new tradable instrument
// using a market framework configuration for a tradable instrument.
func NewTradableInstrument(ctx context.Context, log *logging.Logger, pti *types.TradableInstrument, marketID string, ts common.TimeService, oe products.OracleEngine, broker products.Broker, assetDP uint32) (*TradableInstrument, error) {
	instrument, err := NewInstrument(ctx, log, pti.Instrument, marketID, ts, oe, broker, assetDP)
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
		assetDP:          assetDP, // keep it here for the update call
	}, nil
}

func (i *TradableInstrument) UpdateInstrument(ctx context.Context, log *logging.Logger, ti *types.TradableInstrument, marketID string, oe products.OracleEngine, broker products.Broker) error {
	i.Instrument.Update(ctx, log, ti.Instrument, oe)

	asset := i.Instrument.Product.GetAsset()

	riskModel, err := risk.NewModel(ti.RiskModel, asset)
	if err != nil {
		return fmt.Errorf("unable to instantiate risk model: %w", err)
	}

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
func NewInstrument(ctx context.Context, log *logging.Logger, pi *types.Instrument, marketID string, ts common.TimeService, oe products.OracleEngine, broker products.Broker, assetDP uint32) (*Instrument, error) {
	product, err := products.New(ctx, log, pi.Product, marketID, ts, oe, broker, assetDP)
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

func (i *Instrument) UpdateAuctionState(ctx context.Context, enter bool) {
	i.Product.UpdateAuctionState(ctx, enter)
}

func (i *Instrument) UnsubscribeTradingTerminated(ctx context.Context) {
	i.Product.UnsubscribeTradingTerminated(ctx)
}

func (i *Instrument) UnsubscribeSettlementData(ctx context.Context) {
	i.Product.UnsubscribeSettlementData(ctx)
}

func (i *Instrument) Unsubscribe(ctx context.Context) {
	i.UnsubscribeTradingTerminated(ctx)
	i.UnsubscribeSettlementData(ctx)
}

// NewInstrument will instantiate a new instrument
// using a market framework configuration for a instrument.
func (i *Instrument) Update(ctx context.Context, log *logging.Logger, pi *types.Instrument, oe products.OracleEngine) error {
	if err := i.Product.Update(ctx, pi.Product, oe); err != nil {
		return err
	}

	i.ID = pi.ID
	i.Code = pi.Code
	i.Name = pi.Name
	i.Metadata = pi.Metadata
	return nil
}
