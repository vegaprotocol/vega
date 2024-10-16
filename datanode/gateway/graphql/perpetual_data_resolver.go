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
	"errors"
	"math"

	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type perpetualDataResolver VegaResolverRoot

func (p perpetualDataResolver) SeqNum(_ context.Context, obj *vega.PerpetualData) (int, error) {
	if obj.SeqNum > math.MaxInt {
		return 0, errors.New("funding period sequence number is too large")
	}
	return int(obj.SeqNum), nil
}

func (p perpetualDataResolver) InternalCompositePriceType(_ context.Context, obj *vega.PerpetualData) (CompositePriceType, error) {
	if obj.InternalCompositePriceType == types.CompositePriceType_COMPOSITE_PRICE_TYPE_WEIGHTED {
		return CompositePriceTypeCompositePriceTypeWeighted, nil
	} else if obj.InternalCompositePriceType == types.CompositePriceType_COMPOSITE_PRICE_TYPE_MEDIAN {
		return CompositePriceTypeCompositePriceTypeMedian, nil
	} else {
		return CompositePriceTypeCompositePriceTypeLastTrade, nil
	}
}

func (p perpetualDataResolver) InternalCompositePrice(_ context.Context, obj *vega.PerpetualData) (string, error) {
	return obj.InternalCompositePrice, nil
}

func (p perpetualDataResolver) NextInternalCompositePriceCalc(_ context.Context, obj *vega.PerpetualData) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.NextInternalCompositePriceCalc)), nil
}
