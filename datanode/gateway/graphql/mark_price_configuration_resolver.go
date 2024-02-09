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

	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type compositePriceConfigurationResolver VegaResolverRoot

func (*compositePriceConfigurationResolver) MarkPriceSourceWeights(ctx context.Context, obj *vega.CompositePriceConfiguration) ([]string, error) {
	if obj == nil {
		return []string{}, nil
	}
	return obj.SourceWeights, nil
}

func (*compositePriceConfigurationResolver) MarkPriceSourceStalenessTolerance(ctx context.Context, obj *vega.CompositePriceConfiguration) ([]string, error) {
	if obj == nil {
		return []string{}, nil
	}
	return obj.SourceStalenessTolerance, nil
}

func (*compositePriceConfigurationResolver) CompositePriceType(ctx context.Context, obj *vega.CompositePriceConfiguration) (CompositePriceType, error) {
	if obj == nil {
		return CompositePriceTypeCompositePriceTypeLastTrade, nil
	}
	if obj.CompositePriceType == types.CompositePriceType_COMPOSITE_PRICE_TYPE_WEIGHTED {
		return CompositePriceTypeCompositePriceTypeWeighted, nil
	} else if obj.CompositePriceType == types.CompositePriceType_COMPOSITE_PRICE_TYPE_MEDIAN {
		return CompositePriceTypeCompositePriceTypeMedian, nil
	} else {
		return CompositePriceTypeCompositePriceTypeLastTrade, nil
	}
}

func (*compositePriceConfigurationResolver) DecayPower(ctx context.Context, obj *vega.CompositePriceConfiguration) (int, error) {
	if obj == nil {
		return 0, nil
	}
	return int(obj.DecayPower), nil
}

func (*compositePriceConfigurationResolver) DataSourcesSpecBinding(ctx context.Context, obj *vega.CompositePriceConfiguration) ([]*SpecBindingForCompositePrice, error) {
	if obj == nil || obj.DataSourcesSpecBinding == nil {
		return nil, nil
	}
	binding := make([]*SpecBindingForCompositePrice, 0, len(obj.DataSourcesSpecBinding))
	for _, sbfcp := range obj.DataSourcesSpecBinding {
		binding = append(binding, &SpecBindingForCompositePrice{
			PriceSourceProperty: sbfcp.PriceSourceProperty,
		})
	}
	return binding, nil
}
