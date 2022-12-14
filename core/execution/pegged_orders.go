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

package execution

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

type PeggedOrders struct {
	log         *logging.Logger
	timeService TimeService
	// parked list
	parked   []*types.Order
	isParked map[string]struct{}
}

func NewPeggedOrders(log *logging.Logger, ts TimeService) *PeggedOrders {
	return &PeggedOrders{
		log:         log,
		timeService: ts,
		parked:      []*types.Order{},
		isParked:    map[string]struct{}{},
	}
}

func NewPeggedOrdersFromSnapshot(
	log *logging.Logger,
	ts TimeService,
	state *types.PeggedOrdersState,
) *PeggedOrders {
	p := NewPeggedOrders(log, ts)
	p.parked = state.Parked
	for _, v := range p.parked {
		p.isParked[v.ID] = struct{}{}
	}
	return p
}

func (p *PeggedOrders) Changed() bool {
	return true
}

func (p *PeggedOrders) GetState() *types.PeggedOrdersState {
	parkedCopy := make([]*types.Order, 0, len(p.parked))
	for _, v := range p.parked {
		parkedCopy = append(parkedCopy, v.Clone())
	}

	return &types.PeggedOrdersState{
		Parked: parkedCopy,
	}
}

func (p *PeggedOrders) IsParked(id string) bool {
	_, parked := p.isParked[id]
	return parked
}

func (p *PeggedOrders) Park(o *types.Order) {
	o.UpdatedAt = p.timeService.GetTimeNow().UnixNano()
	o.Status = types.OrderStatusParked
	o.Price = num.UintZero()
	o.OriginalPrice = nil

	p.parked = append(p.parked, o)
	p.isParked[o.ID] = struct{}{}
}

func (p *PeggedOrders) Unpark(oid string) {
	for i, po := range p.parked {
		if po.ID == oid {
			// Remove item from slice
			copy(p.parked[i:], p.parked[i+1:])
			p.parked[len(p.parked)-1] = nil
			p.parked = p.parked[:len(p.parked)-1]
			delete(p.isParked, oid)
			return
		}
	}
}

func (p *PeggedOrders) GetParkedByID(id string) *types.Order {
	for _, o := range p.parked {
		if o.ID == id {
			return o
		}
	}
	return nil
}

func (p *PeggedOrders) AmendParked(amended *types.Order) {
	for i, o := range p.parked {
		if o.ID == amended.ID {
			p.parked[i] = amended
			return
		}
	}

	p.log.Panic("tried to amend a non parked order from the parked list", logging.Order(amended))
}

func (p *PeggedOrders) RemoveAllForParty(
	ctx context.Context, party string, status types.OrderStatus,
) (orders []*types.Order, evts []events.Event) {
	n := 0
	now := p.timeService.GetTimeNow().UnixNano()

	// then we look at the parked and delete + create events
	for _, o := range p.parked {
		if o.Party == party /* && o.Status == types.Order_STATUS_PARKED */ {
			o.UpdatedAt = now
			o.Status = status
			orders = append(orders, o)
			evts = append(evts, events.NewOrderEvent(ctx, o))
			delete(p.isParked, o.ID)
			continue
		}
		// here we insert back in the slice
		p.parked[n] = o
		n++
	}
	p.parked = p.parked[:n]

	return
}

func (p *PeggedOrders) EnterAuction(ctx context.Context) []events.Event {
	var (
		n    = 0
		now  = p.timeService.GetTimeNow().UnixNano()
		evts = []events.Event{}
	)

	for _, o := range p.parked {
		if o.TimeInForce == types.OrderTimeInForceGFN {
			o.UpdatedAt = now
			o.Status = types.OrderStatusCancelled
			evts = append(evts, events.NewOrderEvent(ctx, o))
			delete(p.isParked, o.ID)
			continue
		}

		// here we insert back in the slice
		p.parked[n] = o
		n++
	}

	p.parked = p.parked[:n]

	return evts
}

func (p *PeggedOrders) GetParkedIDs() []string {
	ids := make([]string, 0, len(p.isParked))
	for k := range p.isParked {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	return ids
}

func (p *PeggedOrders) GetAllParkedForParty(party string) (orders []*types.Order) {
	for _, order := range p.parked {
		if order.Party == party {
			orders = append(orders, order)
		}
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	return
}

func (p *PeggedOrders) Settled() []*types.Order {
	// now we can remove the pegged orders too
	peggedOrders := make([]*types.Order, 0, len(p.parked))
	for _, v := range p.parked {
		order := v.Clone()
		order.Status = types.OrderStatusStopped
		peggedOrders = append(peggedOrders, order)
	}
	sort.Slice(peggedOrders, func(i, j int) bool {
		return peggedOrders[i].ID < peggedOrders[j].ID
	})

	p.parked = nil
	p.isParked = map[string]struct{}{}
	return peggedOrders
}

func (p *PeggedOrders) GetParkedOrdersCount() int {
	return len(p.isParked)
}
