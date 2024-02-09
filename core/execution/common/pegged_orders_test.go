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

package common_test

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
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
	p := common.NewPeggedOrders(logging.NewTestLogger(), tm)
	tm.EXPECT().GetTimeNow().AnyTimes()

	// Test empty
	s := p.GetState()
	a.Equal(
		&types.PeggedOrdersState{
			Parked: []*types.Order{},
		},
		s,
	)

	testOrders := getTestOrders()[:4]

	// Test after adding orders
	p.Park(testOrders[0])
	p.Park(testOrders[1])
	p.Park(testOrders[2])
	p.Park(testOrders[3])
	a.Equal(testOrders[0].ID, p.GetState().Parked[0].ID)
	a.Equal(testOrders[1].ID, p.GetState().Parked[1].ID)
	a.Equal(testOrders[2].ID, p.GetState().Parked[2].ID)
	a.Equal(testOrders[3].ID, p.GetState().Parked[3].ID)

	// Test amend
	p.AmendParked(testOrders[0])
	a.True(p.Changed())
	a.Equal(testOrders[0], p.GetState().Parked[0])

	// Test unpark
	p.Unpark(testOrders[3].ID)
	a.Equal(3, len(p.GetState().Parked))
	a.Equal(testOrders[0].ID, p.GetState().Parked[0].ID)
	a.Equal(testOrders[1].ID, p.GetState().Parked[1].ID)
	a.Equal(testOrders[2].ID, p.GetState().Parked[2].ID)

	// Test get functions won't change state
	p.GetAllParkedForParty("party-1")
	p.GetParkedIDs()
	p.GetParkedByID("id-2")
	p.GetParkedOrdersCount()

	// Test restore state
	s = p.GetState()

	newP := common.NewPeggedOrdersFromSnapshot(logging.NewTestLogger(), tm, s)
	a.Equal(s, newP.GetState())
	a.Equal(len(p.GetParkedIDs()), len(newP.GetParkedIDs()))
}
