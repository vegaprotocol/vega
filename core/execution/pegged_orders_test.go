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

package execution_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/execution/mocks"
	"code.vegaprotocol.io/vega/core/libs/crypto"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func getTestOrders() []*types.Order {
	o := []*types.Order{}

	for i := 0; i < 10; i++ {
		p, _ := num.UintFromString(fmt.Sprintf("%d", i*10), 10)

		o = append(o, &types.Order{
			ID:          crypto.RandomHash(),
			MarketID:    "market-1",
			Party:       "party-1",
			Side:        types.SideBuy,
			Price:       p,
			Size:        uint64(i),
			Remaining:   uint64(i),
			TimeInForce: types.OrderTimeInForceFOK,
			Type:        types.OrderTypeMarket,
			Status:      types.OrderStatusActive,
			Reference:   "ref-1",
			Version:     uint64(2),
			BatchID:     uint64(4),
		})
	}

	return o
}

func TestPeggedOrders(t *testing.T) {
	t.Run("snapshot ", testPeggedOrdersSnapshot)
}

func testPeggedOrdersSnapshot(t *testing.T) {
	a := assert.New(t)

	t.Helper()
	ctrl := gomock.NewController(t)
	tm := mocks.NewMockTimeService(ctrl)
	p := execution.NewPeggedOrders(tm)
	a.False(p.Changed())

	// Test empty
	s := p.GetState()
	a.False(p.Changed())
	a.Equal([]*types.Order{}, s)

	testOrders := getTestOrders()[:4]

	// Test after adding orders
	p.Add(testOrders[0])
	p.Add(testOrders[1])
	p.Add(testOrders[2])
	p.Add(testOrders[3])
	a.True(p.Changed())
	a.Equal(testOrders, p.GetState())
	a.False(p.Changed())

	// Test amend
	p.Amend(testOrders[0])
	a.True(p.Changed())
	a.Equal(testOrders, p.GetState())
	a.False(p.Changed())

	// Test park
	tm.EXPECT().GetTimeNow().AnyTimes()
	p.Park(testOrders[1])
	a.True(p.Changed())
	a.Equal(testOrders, p.GetState())
	a.False(p.Changed())

	// Test remove
	p.Remove(testOrders[3])
	testOrders = testOrders[:3]
	a.True(p.Changed())
	a.Equal(testOrders, p.GetState())
	a.False(p.Changed())

	// Test get functions won't change state
	p.GetAllActiveOrders()
	p.GetAllForParty("party-1")
	p.GetByID("id-2")
	a.False(p.Changed())

	// Test restore state
	s = p.GetState()

	ob := matching.NewCachedOrderBook(logging.NewTestLogger(), config.NewDefaultConfig().Execution.Matching, "market-1", false)
	pl := &types.Payload{
		Data: &types.PayloadMatchingBook{
			MatchingBook: &types.MatchingBook{
				MarketID:        "market-1",
				Buy:             testOrders,
				Sell:            nil,
				LastTradedPrice: num.NewUint(100),
				Auction:         false,
				BatchID:         1,
			},
		},
	}
	ob.LoadState(context.Background(), pl)

	newP := execution.NewPeggedOrdersFromSnapshot(s, tm)
	newP.ReconcileWithOrderBook(ob)
	a.Equal(s, newP.GetState())
	a.Equal(len(p.GetAll()), len(newP.GetAll()))

	// if market is in a auction we'll have pegged orders on the market but not the orderbook
	ob2 := matching.NewCachedOrderBook(logging.NewTestLogger(), config.NewDefaultConfig().Execution.Matching, "market-1", false)
	newP2 := execution.NewPeggedOrdersFromSnapshot(s, tm)
	tm.EXPECT().GetTimeNow().AnyTimes()
	for _, o := range newP2.GetAll() {
		newP2.Park(o)
	}
	newP2.ReconcileWithOrderBook(ob2)
	a.Equal(len(p.GetAll()), len(newP2.GetAll()))
}
