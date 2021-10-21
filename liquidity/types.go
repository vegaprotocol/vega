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
		return num.DecimalZero()
	}

	n := num.Zero()
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

type SnapshotablePendingProvisions struct {
	m       map[string]struct{}
	updated bool
}

func (s *SnapshotablePendingProvisions) HasUpdates() bool {
	return s.updated
}

func (s *SnapshotablePendingProvisions) Add(key string) {
	s.updated = true
	s.m[key] = struct{}{}
}

func (s *SnapshotablePendingProvisions) Delete(key string) {
	s.updated = true
	delete(s.m, key)
}

func (s *SnapshotablePendingProvisions) ResetUpdated() {
	s.updated = false
}

func (s *SnapshotablePendingProvisions) Exists(key string) bool {
	_, ok := s.m[key]
	return ok
}

// Provisions is a map of parties to *types.LiquidityProvision.
type ProvisionsPerParty map[string]*types.LiquidityProvision

type SnapshotableProvisionsPerParty struct {
	ProvisionsPerParty
	updated bool
}

func (s *SnapshotableProvisionsPerParty) HasUpdates() bool {
	return s.updated
}

func (s *SnapshotableProvisionsPerParty) ResetUpdated() {
	s.updated = false
}

func (s *SnapshotableProvisionsPerParty) Delete(key string) {
	s.updated = true
	delete(s.ProvisionsPerParty, key)
}

func (s *SnapshotableProvisionsPerParty) Get(key string) (*types.LiquidityProvision, bool) {
	p, ok := s.ProvisionsPerParty[key]
	return p, ok
}

func (s *SnapshotableProvisionsPerParty) Set(key string, p *types.LiquidityProvision) {
	s.updated = true
	s.ProvisionsPerParty[key] = p
}

// Slice returns the parties as a slice.
func (l ProvisionsPerParty) Slice() LiquidityProvisions {
	slice := make(LiquidityProvisions, 0, len(l))
	for _, p := range l {
		slice = append(slice, p)
	}
	// sorting by partyId to ensure any processing in a deterministic manner later on
	sort.Slice(slice, func(i, j int) bool { return slice[i].Party < slice[j].Party })
	return slice
}

func (l ProvisionsPerParty) FeeForTarget(v *num.Uint) num.Decimal {
	return l.Slice().sortByFee().feeForTarget(v)
}

// TotalStake returns the sum of all CommitmentAmount, which corresponds to the
// total stake of a market.
func (l ProvisionsPerParty) TotalStake() *num.Uint {
	n := num.Zero()
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

// ByParty returns the orders grouped by it's PartyID.
func (ords Orders) ByParty() []PartyOrders {
	// first extract all orders, per party
	parties := map[string][]*types.Order{}
	for _, order := range ords {
		parties[order.Party] = append(parties[order.Party], order)
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

type PartiesOrders struct {
	m       map[string]map[string]*types.Order
	updated bool
}

func (o *PartiesOrders) HasUpdates() bool {
	return o.updated
}

func (o *PartiesOrders) ResetUpdated() {
	o.updated = false
}

func (o *PartiesOrders) Get(party, orderID string) (*types.Order, bool) {
	orders, ok := o.m[party]
	if !ok {
		return nil, false
	}
	order, ok := orders[orderID]
	return order, ok
}

// GetForParty expects to read through them, not do any write operation
func (o *PartiesOrders) GetForParty(
	party string) (map[string]*types.Order, bool) {
	orders, ok := o.m[party]
	return orders, ok
}

func (o *PartiesOrders) Add(party string, order *types.Order) {
	o.updated = true
	orders, ok := o.m[party]
	if !ok {
		orders = map[string]*types.Order{}
		o.m[party] = orders
	}
	orders[order.ID] = order
}

func (o *PartiesOrders) Delete(party, order string) {
	o.updated = true
	delete(o.m[party], order)
	if len(o.m[party]) <= 0 {
		delete(o.m, party)
	}
}

func (o *PartiesOrders) DeleteParty(party string) {
	o.updated = true
	delete(o.m, party)
}

func (o *PartiesOrders) ResetForParty(party string) {
	o.updated = true
	o.m[party] = map[string]*types.Order{}
}
