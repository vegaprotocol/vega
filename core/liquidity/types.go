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

package liquidity

import (
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
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

// Provisions provides convenience functions to a slice of *vega/proto.LiquidityProvision.
type Provisions []*types.LiquidityProvision

// feeForTarget returns the right fee given a group of sorted (by ascending fee) LiquidityProvisions.
// To find the right fee we need to find smallest index k such that:
// [target stake] < sum from i=1 to k of [MM-stake-i]. In other words we want in this
// ordered list to find the liquidity providers that supply the liquidity
// that's required. If no such k exists we set k=N.
func (l Provisions) feeForTarget(t *num.Uint) num.Decimal {
	if len(l) == 0 {
		return num.DecimalZero()
	}

	n := num.UintZero()
	for _, i := range l {
		n.AddSum(i.CommitmentAmount)
		if n.GTE(t) {
			return i.Fee
		}
	}

	// return the last one
	return l[len(l)-1].Fee
}

type lpsByFee Provisions

func (l lpsByFee) Len() int           { return len(l) }
func (l lpsByFee) Less(i, j int) bool { return l[i].Fee.LessThan(l[j].Fee) }
func (l lpsByFee) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// sortByFee sorts in-place and returns the LiquidityProvisions for convenience.
func (l Provisions) sortByFee() Provisions {
	byFee := lpsByFee(l)
	sort.Sort(byFee)
	return Provisions(byFee)
}

type SnapshotablePendingProvisions struct {
	m map[string]struct{}
}

func newSnapshotablePendingProvisions() *SnapshotablePendingProvisions {
	return &SnapshotablePendingProvisions{
		m: map[string]struct{}{},
	}
}

func (s *SnapshotablePendingProvisions) Add(key string) {
	s.m[key] = struct{}{}
}

func (s *SnapshotablePendingProvisions) Delete(key string) {
	delete(s.m, key)
}

func (s *SnapshotablePendingProvisions) Exists(key string) bool {
	_, ok := s.m[key]
	return ok
}

// Provisions is a map of parties to *types.LiquidityProvision.
type ProvisionsPerParty map[string]*types.LiquidityProvision

type SnapshotableProvisionsPerParty struct {
	ProvisionsPerParty
}

func newSnapshotableProvisionsPerParty() *SnapshotableProvisionsPerParty {
	return &SnapshotableProvisionsPerParty{
		ProvisionsPerParty: map[string]*types.LiquidityProvision{},
	}
}

func (s *SnapshotableProvisionsPerParty) Delete(key string) {
	delete(s.ProvisionsPerParty, key)
}

func (s *SnapshotableProvisionsPerParty) Get(key string) (*types.LiquidityProvision, bool) {
	p, ok := s.ProvisionsPerParty[key]
	return p, ok
}

func (s *SnapshotableProvisionsPerParty) Set(key string, p *types.LiquidityProvision) {
	s.ProvisionsPerParty[key] = p
}

// Slice returns the parties as a slice.
func (l ProvisionsPerParty) Slice() Provisions {
	slice := make(Provisions, 0, len(l))
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
	n := num.UintZero()
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

type SnapshotablePartiesOrders struct {
	m map[string]map[string]*types.Order
}

func newSnapshotablePartiesOrders() *SnapshotablePartiesOrders {
	return &SnapshotablePartiesOrders{
		m: map[string]map[string]*types.Order{},
	}
}

func (o *SnapshotablePartiesOrders) Get(party, orderID string) (*types.Order, bool) {
	orders, ok := o.m[party]
	if !ok {
		return nil, false
	}
	order, ok := orders[orderID]
	return order, ok
}

// GetForParty expects to read through them, not do any write operation.
func (o *SnapshotablePartiesOrders) GetForParty(
	party string,
) (map[string]*types.Order, bool) {
	orders, ok := o.m[party]
	return orders, ok
}

func (o *SnapshotablePartiesOrders) Add(party string, order *types.Order) {
	orders, ok := o.m[party]
	if !ok {
		orders = map[string]*types.Order{}
		o.m[party] = orders
	}
	orders[order.ID] = order
}

func (o *SnapshotablePartiesOrders) Delete(party, order string) {
	delete(o.m[party], order)
	if len(o.m[party]) <= 0 {
		delete(o.m, party)
	}
}

func (o *SnapshotablePartiesOrders) DeleteParty(party string) {
	delete(o.m, party)
}

func (o *SnapshotablePartiesOrders) ResetForParty(party string) {
	o.m[party] = map[string]*types.Order{}
}
