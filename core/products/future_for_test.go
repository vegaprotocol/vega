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

	"code.vegaprotocol.io/vega/libs/num"

	"code.vegaprotocol.io/vega/core/oracles"
)

func (f *Future) SetSettlementData(ctx context.Context, priceName string, settlementData *num.Numeric) {
	od := oracles.OracleData{Data: map[string]string{}}
	if settlementData.IsUint() {
		od.Data[priceName] = settlementData.Uint().String()
	} else {
		od.Data[priceName] = settlementData.Decimal().String()
	}
	f.updateSettlementData(ctx, od)
}
