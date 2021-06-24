package liquidity

import (
	"sort"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type ToCancel struct {
	Party    string
	OrderIDs []string
}

func (c *ToCancel) Merge(oth *ToCancel) *ToCancel {
	if c.Party != oth.Party {
		panic("could not merge ToCancel from different parties")
	}
	return &ToCancel{
		Party:    c.Party,
		OrderIDs: append(c.OrderIDs, oth.OrderIDs...),
	}
}

func (c *ToCancel) Add(id string) {
	c.OrderIDs = append(c.OrderIDs, id)
}

func (c *ToCancel) Empty() bool {
	return len(c.OrderIDs) <= 0
}

// LiquidityProvisions provides convenience functions to a slice of *vega/proto.LiquidityProvision.
type LiquidityProvisions []*types.LiquidityProvision

// feeForTarget returns the right fee given a group of sorted (by ascending fee) LiquidityProvisions.
// To find the right fee we need to find smallest index k such that:
// [target stake] < sum from i=1 to k of [MM-stake-i]. In other words we want in this
// ordered list to find the liquidity providers that supply the liquidity
// that's required. If no such k exists we set k=N.
func (l LiquidityProvisions) feeForTarget(t *num.Uint) num.Decimal {
	if len(l) == 0 {
		return num.DecimalFromFloat(0)
	}

	n := num.NewUint(0)
	for _, i := range l {
		n.AddSum(i.CommitmentAmount)
		if n.GTE(t) {
			return i.Fee
		}
	}

	// return the last one
	return l[len(l)-1].Fee
}

type lpsByFee LiquidityProvisions

func (l lpsByFee) Len() int           { return len(l) }
func (l lpsByFee) Less(i, j int) bool { return l[i].Fee.LessThan(l[j].Fee) }
func (l lpsByFee) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// sortByFee sorts in-place and returns the LiquidityProvisions for convenience.
func (l LiquidityProvisions) sortByFee() LiquidityProvisions {
	byFee := lpsByFee(l)
	sort.Sort(byFee)
	return LiquidityProvisions(byFee)
}

// Provisions is a map of parties to *types.LiquidityProvision
type ProvisionsPerParty map[string]*types.LiquidityProvision

// Slice returns the parties as a slice.
func (l ProvisionsPerParty) Slice() LiquidityProvisions {
	slice := make(LiquidityProvisions, 0, len(l))
	for _, p := range l {
		slice = append(slice, p)
	}
	// sorting by partyId to ensure any processing in a deterministic manner later on
	sort.Slice(slice, func(i, j int) bool { return slice[i].PartyId < slice[j].PartyId })
	return slice
}

func (l ProvisionsPerParty) FeeForTarget(v *num.Uint) num.Decimal {
	return l.Slice().sortByFee().feeForTarget(v)
}

// TotalStake returns the sum of all CommitmentAmount, which corresponds to the
// total stake of a market.
func (l ProvisionsPerParty) TotalStake() *num.Uint {
	n := num.NewUint(0)
	for _, p := range l {
		n.AddSum(p.CommitmentAmount)
	}
	return n
}

// Orders provides convenience functions to a slice of *veaga/proto.Orders.
type Orders []*types.Order

type PartyOrders struct {
	Party  string
	Orders []*types.Order
}

// ByParty returns the orders grouped by it's PartyID
func (ords Orders) ByParty() []PartyOrders {
	// first extract all orders, per party
	parties := map[string][]*types.Order{}
	for _, order := range ords {
		parties[order.PartyId] = append(parties[order.PartyId], order)
	}

	// now, move stuff from the map, into the PartyOrders type, and sort it
	partyOrders := make([]PartyOrders, 0, len(parties))
	for k, v := range parties {
		partyOrders = append(partyOrders, PartyOrders{k, v})
	}

	// now sort them to guaranty deterministic
	sort.Slice(partyOrders, func(i, j int) bool {
		return partyOrders[i].Party < partyOrders[j].Party
	})
	return partyOrders
}
