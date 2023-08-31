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

package positions

import (
	"context"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"golang.org/x/exp/maps"
)

// Broker (no longer need to mock this, use the broker/mocks wrapper).
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// Engine represents the positions engine.
type Engine struct {
	marketID string
	log      *logging.Logger
	Config

	cfgMu sync.Mutex
	// partyID -> MarketPosition
	positions map[string]*MarketPosition

	// this is basically tracking all position to
	// not perform a copy when positions a retrieved by other engines
	// the pointer is hidden behind the interface, and do not expose
	// any function to mutate them, so we can consider it safe to return
	// this slice.
	positionsCpy []events.MarketPosition

	// keep track of the position updated during the current block to avoid sending
	updatedPositions map[string]struct{}
	positionUpdated  func(context.Context, *MarketPosition)

	broker Broker

	// keep track of open, but distressed positions, this speeds things up when creating the event data
	// and when generating snapshots
	distressedPos map[string]struct{}

	// keeps track of the lowest openInterest
	// per party over a whole epoch
	// should be reseted by the market on new epochs
	partiesOpenInterest map[string]*openInterestRecord

	// keep track of the traded volume during the epoch
	// will be reset
	partiesTradedSize map[string]uint64
}

type openInterestRecord struct {
	Latest uint64
	Lowest uint64
}

// RecordLatest will save the new openInterest for a party
// but also register it as the lowest if it goes below
// the existing lowest openInterest.
func (o *openInterestRecord) RecordLatest(new uint64) {
	o.Latest = new
	if o.Lowest > new {
		o.Lowest = new
	}
}

// Reset will be used at the end of the epoch,
// in order to set the lowest for the epoch to
// be the latest of the previous epoch.
func (o *openInterestRecord) Reset() uint64 {
	lowest := o.Lowest
	o.Lowest = o.Latest

	return lowest
}

// New instantiates a new positions engine.
func New(log *logging.Logger, config Config, marketID string, broker Broker) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	e := &Engine{
		marketID:            marketID,
		Config:              config,
		log:                 log,
		positions:           map[string]*MarketPosition{},
		positionsCpy:        []events.MarketPosition{},
		broker:              broker,
		updatedPositions:    map[string]struct{}{},
		distressedPos:       map[string]struct{}{},
		partiesOpenInterest: map[string]*openInterestRecord{},
		partiesTradedSize:   map[string]uint64{},
	}
	e.positionUpdated = e.bufferPosition
	if config.StreamPositionVerbose {
		e.positionUpdated = e.sendPosition
	}

	return e
}

func (e *Engine) FlushPositionEvents(ctx context.Context) {
	if e.StreamPositionVerbose || len(e.updatedPositions) <= 0 {
		return
	}

	e.sendBufferedPosition(ctx)
}

func (e *Engine) bufferPosition(_ context.Context, pos *MarketPosition) {
	e.updatedPositions[pos.partyID] = struct{}{}
}

func (e *Engine) sendBufferedPosition(ctx context.Context) {
	parties := maps.Keys(e.updatedPositions)
	sort.Strings(parties)
	evts := make([]events.Event, 0, len(parties))

	for _, v := range parties {
		// ensure the position exists,
		// party might have been distressed or something
		if pos, ok := e.positions[v]; ok {
			evts = append(evts, events.NewPositionStateEvent(ctx, pos, e.marketID))
		}
	}

	e.broker.SendBatch(evts)
	e.updatedPositions = make(map[string]struct{}, len(e.updatedPositions))
}

func (e *Engine) sendPosition(ctx context.Context, pos *MarketPosition) {
	e.broker.Send(events.NewPositionStateEvent(ctx, pos, e.marketID))
}

func (e *Engine) Hash() []byte {
	// Fields * FieldSize = (8 * 3)
	// Prices = 32 * 2
	output := make([]byte, len(e.positionsCpy)*((8*3)+(32*2)))
	var i int
	for _, p := range e.positionsCpy {
		values := []uint64{
			uint64(p.Size()),
			uint64(p.Buy()),
			uint64(p.Sell()),
		}

		for _, v := range values {
			binary.BigEndian.PutUint64(output[i:], v)
			i += 8
		}

		// Add bytes for VWBuy and VWSell here
		b := p.BuySumProduct().Bytes()
		copy(output[i:], b[:])
		i += 32
		s := p.SellSumProduct().Bytes()
		copy(output[i:], s[:])
		i += 32
	}

	return crypto.Hash(output)
}

// ReloadConf update the internal configuration of the positions engine.
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.cfgMu.Lock()
	e.Config = cfg
	e.cfgMu.Unlock()
}

// RegisterOrder updates the potential positions for a submitted order, as though
// the order were already accepted.
// It returns the updated position.
// The margins+risk engines need the updated position to determine whether the
// order should be accepted.
func (e *Engine) RegisterOrder(ctx context.Context, order *types.Order) *MarketPosition {
	pos, found := e.positions[order.Party]
	if !found {
		pos = NewMarketPosition(order.Party)
		e.positions[order.Party] = pos
		// append the pointer to the slice as well
		e.positionsCpy = append(e.positionsCpy, pos)
		// create the entry in the open interest map
		e.partiesOpenInterest[order.Party] = &openInterestRecord{}
	}

	pos.RegisterOrder(e.log, order)
	e.positionUpdated(ctx, pos)
	return pos
}

// UnregisterOrder undoes the actions of RegisterOrder. It is used when an order
// has been rejected by the Risk Engine, or when an order is amended or canceled.
func (e *Engine) UnregisterOrder(ctx context.Context, order *types.Order) *MarketPosition {
	pos, found := e.positions[order.Party]
	if !found {
		e.log.Panic("could not find position in engine when unregistering order",
			logging.Order(*order))
	}
	pos.UnregisterOrder(e.log, order)
	e.positionUpdated(ctx, pos)
	return pos
}

// AmendOrder unregisters the original order and then registers the newly amended order
// this method is a quicker way of handling separate unregister+register pairs.
func (e *Engine) AmendOrder(ctx context.Context, originalOrder, newOrder *types.Order) *MarketPosition {
	pos, found := e.positions[originalOrder.Party]
	if !found {
		e.log.Panic("could not find position in engine when amending order",
			logging.Order(*originalOrder),
			logging.Order(*newOrder))
	}

	pos.AmendOrder(e.log, originalOrder, newOrder)
	e.positionUpdated(ctx, pos)
	return pos
}

// UpdateNetwork - functionally the same as the Update func, except for ignoring the network
// party in the trade (whether it be buyer or seller). This could be incorporated into the Update
// function, but we know when we're adding network trades, and having this check every time is
// wasteful, and would only serve to add complexity to the Update func, and slow it down.
func (e *Engine) UpdateNetwork(ctx context.Context, trade *types.Trade, passiveOrder *types.Order) []events.MarketPosition {
	// there's only 1 position
	var (
		ok  bool
		pos *MarketPosition
	)
	size := int64(trade.Size)
	if trade.Buyer != "network" {
		pos, ok = e.positions[trade.Buyer]
		if !ok {
			e.log.Panic("could not find buyer position",
				logging.Trade(*trade))
		}

		if pos.buy < int64(trade.Size) {
			e.log.Panic("network trade with a potential buy position < to the trade size",
				logging.PartyID(trade.Buyer),
				logging.Int64("potential-buy", pos.buy),
				logging.Trade(*trade))
		}
		pos.size += size
	} else {
		pos, ok = e.positions[trade.Seller]
		if !ok {
			e.log.Panic("could not find seller position",
				logging.Trade(*trade))
		}

		if pos.sell < int64(trade.Size) {
			e.log.Panic("network trade with a potential sell position < to the trade size",
				logging.PartyID(trade.Seller),
				logging.Int64("potential-sell", pos.sell),
				logging.Trade(*trade))
		}
		// size is negative in case of a sale
		pos.size -= size
	}

	pos.UpdateOnOrderChange(e.log, passiveOrder.Side, passiveOrder.Price, trade.Size, false)
	var posSize uint64
	if pos.size > 0 {
		posSize = uint64(pos.size)
	}
	e.partiesOpenInterest[pos.partyID].RecordLatest(posSize)
	e.partiesTradedSize[pos.partyID] += uint64(size)

	e.positionUpdated(ctx, pos)

	cpy := pos.Clone()
	return []events.MarketPosition{*cpy}
}

// Update pushes the previous positions on the channel + the updated open volumes of buyer/seller.
func (e *Engine) Update(ctx context.Context, trade *types.Trade, passiveOrder, aggressiveOrder *types.Order) []events.MarketPosition {
	buyer, ok := e.positions[trade.Buyer]
	if !ok {
		e.log.Panic("could not find buyer position",
			logging.Trade(*trade))
	}

	seller, ok := e.positions[trade.Seller]
	if !ok {
		e.log.Panic("could not find seller position",
			logging.Trade(*trade))
	}

	// now we check if the trade is possible based on the potential positions
	// this should always be true, no trade can happen without the equivalent
	// potential position
	if buyer.buy < int64(trade.Size) {
		e.log.Panic("trade with a potential buy position < to the trade size",
			logging.PartyID(trade.Buyer),
			logging.Int64("potential-buy", buyer.buy),
			logging.Trade(*trade))
	}
	if seller.sell < int64(trade.Size) {
		e.log.Panic("trade with a potential sell position < to the trade size",
			logging.PartyID(trade.Seller),
			logging.Int64("potential-sell", buyer.sell),
			logging.Trade(*trade))
	}

	// Update long/short actual position for buyer and seller.
	// The buyer's position increases and the seller's position decreases.
	buyer.size += int64(trade.Size)
	seller.size -= int64(trade.Size)

	aggressive := buyer
	passive := seller
	if aggressiveOrder.Side == types.SideSell {
		aggressive = seller
		passive = buyer
	}

	// Update potential positions & vwaps. Potential positions decrease for both buyer and seller.
	aggressive.UpdateOnOrderChange(e.log, aggressiveOrder.Side, aggressiveOrder.Price, trade.Size, false)
	passive.UpdateOnOrderChange(e.log, passiveOrder.Side, passiveOrder.Price, trade.Size, false)

	ret := []events.MarketPosition{
		*buyer.Clone(),
		*seller.Clone(),
	}

	e.positionUpdated(ctx, buyer)
	e.positionUpdated(ctx, seller)

	var buyerSize, sellerSize uint64
	if buyer.size > 0 {
		buyerSize = uint64(buyer.size)
	}
	if seller.size > 0 {
		sellerSize = uint64(seller.size)
	}
	e.partiesOpenInterest[buyer.partyID].RecordLatest(buyerSize)
	e.partiesTradedSize[buyer.partyID] += trade.Size
	e.partiesOpenInterest[seller.partyID].RecordLatest(sellerSize)
	e.partiesTradedSize[seller.partyID] += trade.Size

	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("Positions Updated for trade",
			logging.Trade(*trade),
			logging.String("buyer-position", fmt.Sprintf("%+v", buyer)),
			logging.String("seller-position", fmt.Sprintf("%+v", seller)))
	}
	return ret
}

// RemoveDistressed Removes positions for distressed parties, and returns the most up to date positions we have.
func (e *Engine) RemoveDistressed(parties []events.MarketPosition) []events.MarketPosition {
	ret := make([]events.MarketPosition, 0, len(parties))
	for _, party := range parties {
		e.log.Warn("removing party from positions engine",
			logging.String("party-id", party.Party()))

		party := party.Party()
		if current, ok := e.positions[party]; ok {
			ret = append(ret, current)
		}
		// remove from the map
		delete(e.positions, party)
		delete(e.distressedPos, party)
		delete(e.partiesOpenInterest, party)
		// remove from the slice
		for i := range e.positionsCpy {
			if e.positionsCpy[i].Party() == party {
				e.log.Warn("removing party from positions engine (cpy slice)",
					logging.String("party-id", party))
				e.positionsCpy = append(e.positionsCpy[:i], e.positionsCpy[i+1:]...)

				break
			}
		}
	}
	return ret
}

// UpdateMarkPrice update the mark price on all positions and return a slice
// of the updated positions.
func (e *Engine) UpdateMarkPrice(markPrice *num.Uint) []events.MarketPosition {
	for _, pos := range e.positions {
		pos.price.Set(markPrice)
	}
	return e.positionsCpy
}

func (e *Engine) GetOpenInterest() uint64 {
	openInterest := uint64(0)
	for _, pos := range e.positions {
		if pos.size > 0 {
			openInterest += uint64(pos.size)
		}
	}
	return openInterest
}

func (e *Engine) GetOpenInterestGivenTrades(trades []*types.Trade) uint64 {
	oi := e.GetOpenInterest()
	d := int64(0)
	// Store changes to positions across trades locally
	posLocal := make(map[string]int64)
	var ok bool
	for _, t := range trades {
		if t.Seller == t.Buyer {
			// ignore wash trade
			continue
		}
		bPos, sPos := int64(0), int64(0)
		if bPos, ok = posLocal[t.Buyer]; !ok {
			if p, ok := e.positions[t.Buyer]; ok {
				bPos = p.size
			}
		}
		if sPos, ok = posLocal[t.Seller]; !ok {
			if p, ok := e.positions[t.Seller]; ok {
				sPos = p.size
			}
		}

		bPosNew := bPos + int64(t.Size)
		sPosNew := sPos - int64(t.Size)
		posLocal[t.Buyer] = bPosNew
		posLocal[t.Seller] = sPosNew
		// Change in open interest due to trades equals change in longs for both parties
		d += max(0, bPosNew) - max(0, bPos) + max(0, sPosNew) - max(0, sPos)
	}
	if d < 0 {
		return oi - uint64(-d)
	}
	return oi + uint64(d)
}

//nolint:unparam
func max(a int64, b int64) int64 {
	if a >= b {
		return a
	}
	return b
}

// Positions is just the logic to update buyer, will eventually return the MarketPosition we need to push.
func (e *Engine) Positions() []events.MarketPosition {
	return e.positionsCpy
}

// MarkDistressed - mark any distressed parties as such, returns the IDs of the parties who weren't distressed before.
func (e *Engine) MarkDistressed(closed []string) ([]string, []string) {
	// assume number of distressed parties is roughly equal, it isn't but should overall reduce reallocs
	// create a copy, otherwise newly distressed positions are unmarked
	prevDistressed := make(map[string]struct{}, len(e.distressedPos))
	for k := range e.distressedPos {
		prevDistressed[k] = struct{}{}
	}
	nIDs := make([]string, 0, len(closed))
	for _, c := range closed {
		if _, ok := prevDistressed[c]; ok {
			// this party was already known to be distressed, leave it as-is
			// delete from this map, so we are left with only parties who were distressed, but aren't anymore
			delete(prevDistressed, c)
			continue
		}
		// not in distressed map -> get position, mark as distressed
		e.positions[c].distressed = true
		// add the new distressed party to the map
		e.distressedPos[c] = struct{}{}
		// add the ID to the slice of distressed ID's for the event
		nIDs = append(nIDs, c)
	}
	// if we didn't have any previously distressed parties to update, we're done
	if len(prevDistressed) == 0 {
		return nIDs, nil
	}
	// mark parties who are no longer distressed accoringly
	nd := make([]string, 0, len(prevDistressed))
	for k := range prevDistressed {
		// mark as no longer distressed
		e.positions[k].distressed = false
		// remove from the map
		delete(e.distressedPos, k)
		nd = append(nd, k)
	}
	sort.Strings(nIDs)
	sort.Strings(nd)
	return nIDs, nd
}

// GetPositionByPartyID - return current position for a given party, it's used in margin checks during auctions
// we're not specifying an interface of the return type, and we return a pointer to a copy for the nil.
func (e *Engine) GetPositionByPartyID(partyID string) (*MarketPosition, bool) {
	pos, ok := e.positions[partyID]
	if !ok {
		return nil, false
	}
	cpy := pos.Clone()
	// return a copy
	return cpy, true
}

func (e *Engine) GetPositionsByParty(ids ...string) []events.MarketPosition {
	ret := make([]events.MarketPosition, 0, len(ids))
	for _, id := range ids {
		pos, ok := e.positions[id]
		if ok {
			ret = append(ret, pos.Clone())
		}
	}
	return ret
}

// Parties returns a list of all the parties in the position engine.
func (e *Engine) Parties() []string {
	parties := make([]string, 0, len(e.positions))
	for _, v := range e.positions {
		parties = append(parties, v.Party())
	}
	return parties
}

func (e *Engine) GetClosedPositions() []events.MarketPosition {
	out := []events.MarketPosition{}

	for _, v := range e.positions {
		if v.Closed() {
			e.remove(v)
			out = append(out, v)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Party() < out[j].Party() })

	return out
}

// GetOpenPositionCount returns the number of open positions in the market.
func (e *Engine) GetOpenPositionCount() uint64 {
	var total uint64
	for _, mp := range e.positionsCpy {
		if mp.Size() != 0 {
			total++
		}
	}
	return total
}

// GetPartiesLowestOpenInterestForEpoch will return a map of parties
// and minimal open interest for the epoch, it is meant to be called
// at the end of the epoch, and will reset the lowest open interest
// of the party to the latest open interest recored for the epoch.
func (e *Engine) GetPartiesLowestOpenInterestForEpoch() map[string]uint64 {
	out := map[string]uint64{}

	for party, oi := range e.partiesOpenInterest {
		out[party] = oi.Reset()
	}

	return out
}

// GetPartiesTradedVolumeForEpoch will return a map of parties
// and their traded volume recorded during this epoch.
func (e *Engine) GetPartiesTradedVolumeForEpoch() (out map[string]uint64) {
	out, e.partiesTradedSize = e.partiesTradedSize, map[string]uint64{}

	return out
}

func (e *Engine) remove(p *MarketPosition) {
	// delete from the map first
	delete(e.positions, p.partyID)
	delete(e.partiesOpenInterest, p.partyID)
	delete(e.distressedPos, p.partyID) // in case party was previously flagged as distressed

	// remove from the slice
	for i := range e.positionsCpy {
		if e.positionsCpy[i].Party() == p.partyID {
			e.positionsCpy = append(e.positionsCpy[:i], e.positionsCpy[i+1:]...)
			break
		}
	}
}
