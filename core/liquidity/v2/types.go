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

// sortByFee sorts in-place and returns the LiquidityProvisions for convenience.
func (l Provisions) sortByFee() Provisions {
	sort.Slice(l, func(i, j int) bool { return l[i].Fee.LessThan(l[j].Fee) })
	return l
}

func (pp Provisions) Get(key string) (*types.LiquidityProvision, int) {
	for idx, pp := range pp {
		if pp.Party == key {
			return pp, idx
		}
	}

	return nil, -1
}

func (pp *Provisions) Set(lp *types.LiquidityProvision) {
	_, idx := pp.Get(lp.Party)
	if idx > -1 {
		p := *pp
		p[idx] = lp
		return
	}

	*pp = append(*pp, lp)
}

// ProvisionsPerParty maps parties to *types.LiquidityProvision.
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
	sort.SliceStable(slice, func(i, j int) bool { return slice[i].Party < slice[j].Party })
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

type SnapshotablePendingProvisions struct {
	PendingProvisions Provisions
}

func newSnapshotablePendingProvisions() *SnapshotablePendingProvisions {
	return &SnapshotablePendingProvisions{
		PendingProvisions: Provisions{},
	}
}

func (s SnapshotablePendingProvisions) Slice() Provisions {
	return s.PendingProvisions
}

func (s *SnapshotablePendingProvisions) Delete(key string) {
	_, id := s.PendingProvisions.Get(key)
	if id == -1 {
		return
	}

	s.PendingProvisions = append(s.PendingProvisions[:id], s.PendingProvisions[id+1:]...)
}

func (s *SnapshotablePendingProvisions) Get(key string) (*types.LiquidityProvision, bool) {
	lp, idx := s.PendingProvisions.Get(key)
	return lp, idx > -1
}

func (s *SnapshotablePendingProvisions) Set(lp *types.LiquidityProvision) {
	s.PendingProvisions.Set(lp)
}

func (s *SnapshotablePendingProvisions) Len() int {
	return len(s.PendingProvisions)
}

type sliceRing[T any] struct {
	s   []T
	pos int
}

func restoreSliceRing[T any](s []T, position int) *sliceRing[T] {
	return &sliceRing[T]{
		s:   s,
		pos: position,
	}
}

func NewSliceRing[T any](size uint64) *sliceRing[T] {
	return &sliceRing[T]{
		s:   make([]T, size),
		pos: 0,
	}
}

func (r *sliceRing[T]) Add(val T) {
	if len(r.s) == 0 {
		return
	}

	r.s[r.pos] = val

	if r.pos == cap(r.s)-1 {
		r.pos = 0
		return
	}
	r.pos++
}

func (r *sliceRing[T]) ModifySize(newSize uint64) {
	currentCap := cap(r.s)
	currentCapUint := uint64(currentCap)
	if currentCapUint == newSize {
		return
	}

	newS := make([]T, newSize)

	// decrease
	if newSize < currentCapUint {
		newS = r.s[currentCapUint-newSize:]
		r.s = newS
		r.pos = 0
		return
	}

	// increase
	for i := 0; i < currentCap; i++ {
		newS[i] = r.s[i]
	}

	r.s = newS
	r.pos = currentCap
}

func (r sliceRing[T]) Slice() []T {
	return r.s
}

func (r sliceRing[T]) Len() int {
	return len(r.s)
}

func (r sliceRing[T]) Position() int {
	return r.pos
}
