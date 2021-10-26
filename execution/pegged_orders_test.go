package execution

import (
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

func getTestOrders() []*types.Order {
	o := []*types.Order{}

	for i := 0; i < 10; i++ {
		p, _ := num.UintFromString(fmt.Sprintf("%d", i*10), 10)

		o = append(o, &types.Order{
			ID:          fmt.Sprintf("id-%d", i),
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
	p := NewPeggedOrders()
	a.False(p.changed())

	// Test empty
	s := p.GetState()
	a.False(p.changed())
	a.Equal([]*types.Order{}, s)

	testOrders := getTestOrders()[:4]

	// Test after adding orders
	p.Add(testOrders[0])
	p.Add(testOrders[1])
	p.Add(testOrders[2])
	p.Add(testOrders[3])
	a.True(p.changed())
	a.Equal(testOrders, p.GetState())
	a.False(p.changed())

	// Test amend
	testOrders[0].ID = "id-changed"

	p.Amend(testOrders[0])
	a.True(p.changed())
	a.Equal(testOrders, p.GetState())
	a.False(p.changed())

	// Test park
	p.Park(testOrders[1])
	a.True(p.changed())
	a.Equal(testOrders, p.GetState())
	a.False(p.changed())

	// Test remove
	p.Remove(testOrders[3])
	testOrders = testOrders[:3]
	a.True(p.changed())
	a.Equal(testOrders, p.GetState())
	a.False(p.changed())

	// Test get functions won't change state
	p.GetAllActiveOrders()
	p.GetAllForParty("party-1")
	p.GetByID("id-2")
	a.False(p.changed())

	// Test restore state
	s = p.GetState()

	newP := NewPeggedOrdersFromSnapshot(s)
	a.Equal(s, newP.GetState())
}
