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

	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/libs/num"
)

func (f *Future) SetSettlementData(ctx context.Context, priceName string, settlementData *num.Numeric) {
	od := dscommon.Data{Data: map[string]string{}}
	if settlementData.IsUint() {
		od.Data[priceName] = settlementData.Uint().String()
	} else {
		od.Data[priceName] = settlementData.Decimal().String()
	}
	f.updateSettlementData(ctx, od)
}
