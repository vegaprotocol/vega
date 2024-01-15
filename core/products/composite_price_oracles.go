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
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/libs/num"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/pkg/errors"
)

type CompositePriceOracle struct {
	subID   spec.SubscriptionID
	binding compositePriceOracleBinding
	unsub   spec.Unsubscriber
}

type compositePriceOracleBinding struct {
	priceProperty string
	priceType     datapb.PropertyKey_Type
	priceDecimals uint64
}

func NewCompositePriceOracle(ctx context.Context, oracleEngine OracleEngine, oracleSpec *datasource.Spec, binding *datasource.SpecBindingForCompositePrice, cb spec.OnMatchedData) (*CompositePriceOracle, error) {
	bind, err := newCompositePriceBinding(oracleSpec, binding)
	if err != nil {
		return nil, err
	}

	os, err := spec.New(*datasource.SpecFromDefinition(*oracleSpec.Data, 0))
	if err != nil {
		return nil, err
	}

	cpo := &CompositePriceOracle{
		binding: bind,
	}

	err = cpo.bindCompositePriceSource(ctx, oracleEngine, os, cb)
	if err != nil {
		return nil, err
	}

	return cpo, nil
}

func newCompositePriceBinding(spec *datasource.Spec, binding *datasource.SpecBindingForCompositePrice) (compositePriceOracleBinding, error) {
	priceProp := strings.TrimSpace(binding.PriceSourceProperty)
	if len(priceProp) == 0 {
		return compositePriceOracleBinding{}, errors.New("binding for price source data cannot be blank")
	}
	priceT, dec := getSettleTypeAndDec(spec)

	return compositePriceOracleBinding{
		priceProperty: priceProp,
		priceType:     priceT,
		priceDecimals: dec,
	}, nil
}

func (cpo *CompositePriceOracle) bindCompositePriceSource(ctx context.Context, oe OracleEngine, spec *spec.Spec, cb spec.OnMatchedData) error {
	err := spec.EnsureBoundableProperty(cpo.binding.priceProperty, cpo.binding.priceType)
	if err != nil {
		return fmt.Errorf("invalid oracle spec binding for composite price source data: %w", err)
	}
	if cpo.subID, cpo.unsub, err = oe.Subscribe(ctx, *spec, cb); err != nil {
		return fmt.Errorf("could not subscribe to oracle engine for price source data: %w", err)
	}
	return nil
}

func (cpo *CompositePriceOracle) UnsubAll(ctx context.Context) {
	if cpo.unsub != nil {
		cpo.unsub(ctx, cpo.subID)
		cpo.unsub = nil
	}
}

func (cpo *CompositePriceOracle) GetData(data common.Data) (*num.Numeric, error) {
	priceData := &num.Numeric{}
	switch cpo.binding.priceType {
	case datapb.PropertyKey_TYPE_DECIMAL:
		priceDataAsDecimal, err := data.GetDecimal(cpo.binding.priceProperty)
		if err != nil {
			return nil, err
		}

		priceData.SetDecimal(&priceDataAsDecimal)
	default:
		priceDataAsUint, err := data.GetUint(cpo.binding.priceProperty)
		if err != nil {
			return nil, err
		}

		priceData.SetUint(priceDataAsUint)
	}
	return priceData, nil
}
