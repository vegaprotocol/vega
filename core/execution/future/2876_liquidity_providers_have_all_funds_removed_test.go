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

package future_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssue2876(t *testing.T) {
	now := time.Unix(10, 0)
	// set the range so that the old bounds are reproduced
	lpRange := 0.0714285714
	tm := getTestMarketWithDP(t, now, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 30}, 3, lpRange)
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, vgcrypto.RandomHash())

	tm.market.OnTick(ctx, now)

	addAccountWithAmount(tm, "party-0", 100000000)
	addAccountWithAmount(tm, "party-1", 100000000)
	addAccountWithAmount(tm, "party-2", 100000000)
	addAccountWithAmount(tm, "party-3", 100000000)
	addAccountWithAmount(tm, "party-4", 100000000)

	orders := []*types.Order{
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "opening1", types.SideBuy, "party-3", 10, 3000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFA, "opening2", types.SideBuy, "party-3", 10, 4000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGFA, "opening3", types.SideSell, "party-4", 10, 4000),
		getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "opening4", types.SideSell, "party-4", 10, 5500),
	}
	for _, o := range orders {
		conf, err := tm.market.SubmitOrder(ctx, o)
		require.NotNil(t, conf)
		require.NoError(t, err)
	}
	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "party-0", 20, 3500)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NotNil(t, o1conf)
	require.NoError(t, err)

	o2 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order02", types.SideSell, "party-1", 20, 4000)
	o2conf, err := tm.market.SubmitOrder(ctx, o2)
	require.NotNil(t, o2conf)
	require.NoError(t, err)

	o3 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order03", types.SideBuy, "party-2", 10, 5500)
	o3conf, err := tm.market.SubmitOrder(ctx, o3)
	require.NotNil(t, o3conf)
	require.NoError(t, err)

	o4 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order04", types.SideSell, "party-2", 10, 5000)
	o4conf, err := tm.market.SubmitOrder(ctx, o4)
	require.NotNil(t, o4conf)
	require.NoError(t, err)

	lporder := types.LiquidityProvisionSubmission{
		MarketID:         tm.market.GetID(),
		CommitmentAmount: num.NewUint(1000000),
		Fee:              num.DecimalFromFloat(0.01),
	}

	err = tm.market.SubmitLiquidityProvision(ctx, &lporder, "party-2", vgcrypto.RandomHash())
	assert.NoError(t, err)

	bondAccount, err := tm.collateralEngine.GetOrCreatePartyBondAccount(ctx, "party-2", tm.market.GetID(), tm.asset)
	assert.NoError(t, err)
	// we expect the whole commitment to be there
	assert.True(t, bondAccount.Balance.EQ(num.NewUint(1000000)))

	// but also some margin to cover the orders
	marginAccount, err := tm.collateralEngine.GetPartyMarginAccount(tm.market.GetID(), "party-2", tm.asset)
	assert.NoError(t, err)
	assert.True(t, marginAccount.Balance.EQ(num.NewUint(15000)))

	// but also some funds left in the genearal
	generalAccount, err := tm.collateralEngine.GetPartyGeneralAccount("party-2", tm.asset)
	assert.NoError(t, err)
	assert.True(t, generalAccount.Balance.EQ(num.NewUint(98985000)))
}
