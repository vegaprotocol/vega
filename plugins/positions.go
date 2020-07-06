package plugins

import (
	"context"
	"math"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

	"github.com/pkg/errors"
)

var (
	ErrMarketNotFound = errors.New("could not find market")
	ErrPartyNotFound  = errors.New("party not found")
)

// SPE SettlePositionEvent
type SPE interface {
	events.Event
	PartyID() string
	MarketID() string
	Price() uint64
	Trades() events.TradeSettlement
}

// SDE SettleDistressedEvent
type SDE interface {
	events.Event
	PartyID() string
	MarketID() string
	Margin() uint64
	Price() uint64
}

// LSE LossSocializationEvent
type LSE interface {
	events.Event
	PartyID() string
	MarketID() string
	Amount() int64
	AmountLost() int64
}

// PosBuffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/pos_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins PosBuffer
type PosBuffer interface {
	Subscribe() (<-chan []events.SettlePosition, int)
	Unsubscribe(int)
}

// LossSocializationBuffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/loss_socialization_buffer_mock.go -package mocks code.vegaprotocol.io/vega/plugins LossSocializationBuffer
type LossSocializationBuffer interface {
	Subscribe() (<-chan []events.LossSocialization, int)
	Unsubscribe(int)
}

// Positions plugin taking settlement data to build positions API data
type Positions struct {
	*subscribers.Base
	mu    *sync.RWMutex
	buf   PosBuffer
	lsbuf LossSocializationBuffer
	ref   int
	lsref int
	ch    <-chan []events.SettlePosition
	lsch  <-chan []events.LossSocialization
	data  map[string]map[string]Position
}

func NewPositions(buf PosBuffer, lsbuf LossSocializationBuffer) *Positions {
	ctx := context.TODO()
	return &Positions{
		Base:  subscribers.NewBase(ctx, 10, true),
		mu:    &sync.RWMutex{},
		data:  map[string]map[string]Position{},
		buf:   buf,
		lsbuf: lsbuf,
	}
}

func (p *Positions) Push(e events.Event) {
	switch te := e.(type) {
	case SPE:
		p.updatePosition(te)
	case SDE:
		p.updateSettleDestressed(te)
	case LSE:
		p.applyLossSocializationEvent(te)
	}
}

func (p *Positions) Start(ctx context.Context) {
	p.mu.Lock()
	if p.ch == nil && p.lsch == nil {
		// get the channel and the reference
		p.ch, p.ref = p.buf.Subscribe()
		p.lsch, p.lsref = p.lsbuf.Subscribe()
		// start consuming the data
		go p.consume(ctx)
	}
	p.mu.Unlock()
}

func (p *Positions) Stop() {
	p.mu.Lock()
	if p.ch != nil {
		// only unsubscribe if ch was set, otherwise we might end up unregistering ref 0, which
		// could (in theory at least) be used by another component
		p.buf.Unsubscribe(p.ref)
		p.ref = 0

		p.lsbuf.Unsubscribe(p.lsref)
		p.lsref = 0
	}
	// we don't need to reassign ch here, because the channel is closed, the consume routine
	// will pick up on the fact that we don't have to consume data anylonger, and the ch/ref fields
	// will be unset there
	p.mu.Unlock()
}

// consume keep reading the channel for as long as we need to
func (p *Positions) consume(ctx context.Context) {
	defer func() {
		p.Stop()
		p.ch = nil
		p.lsch = nil
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case evts, open := <-p.lsch:
			if !open {
				return
			}
			p.mu.Lock()
			p.applyLossSocialization(evts)
			p.mu.Unlock()
		case update, open := <-p.ch:
			if !open {
				return
			}
			p.mu.Lock()
			p.updateData(update)
			p.mu.Unlock()
		}
	}
}

func (p *Positions) applyLossSocializationEvent(e LSE) {
	p.mu.Lock()
	marketID, partyID, amountLoss := e.MarketID(), e.PartyID(), e.AmountLost()
	pos, ok := p.data[marketID][partyID]
	if !ok {
		return
	}
	if amountLoss < 0 {
		pos.loss += float64(-amountLoss)
	} else {
		pos.adjustment += float64(amountLoss)
	}
	pos.RealisedPNLFP += float64(amountLoss)
	pos.RealisedPNL += amountLoss
	p.data[marketID][partyID] = pos
	p.mu.Unlock()
}

func (p *Positions) applyLossSocialization(evts []events.LossSocialization) {
	for _, evt := range evts {
		marketID, partyID, amountLoss := evt.MarketID(), evt.PartyID(), evt.AmountLost()
		pos, ok := p.data[marketID][partyID]
		if !ok {
			// do nothing, market/party does not exists, but that should not happen
			continue
		}

		// amountLoss will be negative for a good trader, as they lost monies because of bad trader
		// inverse is true for the bad trader as they kind of stole monies from the network
		if amountLoss < 0 {
			// good trader
			pos.loss += float64(-amountLoss)
		} else {
			// bad trader
			pos.adjustment += float64(amountLoss)
		}
		pos.RealisedPNLFP += float64(amountLoss)
		pos.RealisedPNL += amountLoss

		p.data[marketID][partyID] = pos
	}
}

func (p *Positions) updatePosition(e SPE) {
	p.mu.Lock()
	mID, tID := e.MarketID(), e.PartyID()
	if _, ok := p.data[mID]; !ok {
		p.data[mID] = map[string]Position{}
	}
	calc, ok := p.data[mID][tID]
	if !ok {
		calc = speToProto(e)
	}
	updateSettlePosition(&calc, e)
	p.data[mID][tID] = calc
	p.mu.Unlock()
}

func (p *Positions) updateData(raw []events.SettlePosition) {
	for _, sp := range raw {
		if sp == nil {
			continue
		}
		mID, tID := sp.MarketID(), sp.Party()
		if _, ok := p.data[mID]; !ok {
			p.data[mID] = map[string]Position{}
		}
		calc, ok := p.data[mID][tID]
		if !ok {
			calc = evtToProto(sp)
		}
		updatePosition(&calc, sp)
		p.data[mID][tID] = calc
	}
}

// GetPositionsByMarketAndParty get the position of a single trader in a given market
func (p *Positions) GetPositionsByMarketAndParty(market, party string) (*types.Position, error) {
	p.mu.RLock()
	mp, ok := p.data[market]
	if !ok {
		p.mu.RUnlock()
		return nil, nil
	}
	pos, ok := mp[party]
	if !ok {
		p.mu.RUnlock()
		return nil, nil
	}
	p.mu.RUnlock()
	return &pos.Position, nil
}

// GetPositionsByParty get all positions for a given trader
func (p *Positions) GetPositionsByParty(party string) ([]*types.Position, error) {
	p.mu.RLock()
	// at most, trader is active in all markets
	positions := make([]*types.Position, 0, len(p.data))
	for _, traders := range p.data {
		if pos, ok := traders[party]; ok {
			positions = append(positions, &pos.Position)
		}
	}
	p.mu.RUnlock()
	if len(positions) == 0 {
		return nil, nil
		// return nil, ErrPartyNotFound
	}
	return positions, nil
}

// GetPositionsByMarket get all trader positions in a given market
func (p *Positions) GetPositionsByMarket(market string) ([]*types.Position, error) {
	p.mu.RLock()
	mp, ok := p.data[market]
	if !ok {
		p.mu.RUnlock()
		return nil, ErrMarketNotFound
	}
	s := make([]*types.Position, 0, len(mp))
	for _, tp := range mp {
		s = append(s, &tp.Position)
	}
	p.mu.RUnlock()
	return s, nil
}

func calculateOpenClosedVolume(currentOpenVolume, tradedVolume int64) (int64, int64) {
	if currentOpenVolume != 0 && ((currentOpenVolume > 0) != (tradedVolume > 0)) {
		var closedVolume int64
		if absUint64(tradedVolume) > absUint64(currentOpenVolume) {
			closedVolume = currentOpenVolume
		} else {
			closedVolume = -tradedVolume
		}
		return tradedVolume + closedVolume, closedVolume
	}
	return tradedVolume, 0
}

func closeV(p *Position, closedVolume int64, tradedPrice uint64) float64 {
	if closedVolume == 0 {
		return 0
	}
	realisedPnlDelta := float64(closedVolume) * (float64(tradedPrice) - p.AverageEntryPriceFP)
	p.RealisedPNLFP += realisedPnlDelta
	p.OpenVolume -= closedVolume
	return realisedPnlDelta
}

func updateVWAP(vwap float64, volume int64, addVolume int64, addPrice uint64) float64 {
	if volume+addVolume == 0 {
		return 0
	}
	return float64(((vwap * float64(volume)) + (float64(addPrice) * float64(addVolume))) / (float64(volume) + float64(addVolume)))
}

func openV(p *Position, openedVolume int64, tradedPrice uint64) {
	// calculate both average entry price here.
	p.AverageEntryPriceFP = updateVWAP(p.AverageEntryPriceFP, p.OpenVolume, openedVolume, tradedPrice)
	p.OpenVolume += openedVolume
}

func mtm(p *Position, markPrice uint64) {
	if p.OpenVolume == 0 {
		p.UnrealisedPNLFP = 0
		p.UnrealisedPNL = 0
		return
	}
	p.UnrealisedPNLFP = float64(p.OpenVolume) * (float64(markPrice) - p.AverageEntryPriceFP)
}

func updateSettlePosition(p *Position, e SPE) {
	for _, t := range e.Trades() {
		openedVolume, closedVolume := calculateOpenClosedVolume(p.OpenVolume, t.Size())
		_ = closeV(p, closedVolume, t.Price())
		openV(p, openedVolume, t.Price())
		p.AverageEntryPrice = uint64(math.Round(p.AverageEntryPriceFP))
		p.RealisedPNL = int64(math.Round(p.RealisedPNLFP))
	}
	mtm(p, e.Price())
	p.UnrealisedPNL = int64(math.Round(p.UnrealisedPNLFP))
}

func (p *Positions) updateSettleDestressed(e SDE) {
	p.mu.Lock()
	mID, tID := e.MarketID(), e.PartyID()
	if _, ok := p.data[mID]; !ok {
		p.data[mID] = map[string]Position{}
	}
	calc, ok := p.data[mID][tID]
	if !ok {
		calc = speToProto(e)
	}
	margin := e.Margin()
	calc.RealisedPNL += calc.UnrealisedPNL
	calc.RealisedPNLFP += calc.UnrealisedPNLFP
	calc.OpenVolume = 0
	calc.UnrealisedPNL = 0
	calc.AverageEntryPrice = 0
	// realised P&L includes whatever we had in margin account at this point
	calc.RealisedPNL -= int64(margin)
	calc.RealisedPNLFP -= float64(margin)
	// @TODO average entry price shouldn't be affected(?)
	// the volume now is zero, though, so we'll end up moving this position to storage
	calc.UnrealisedPNLFP = 0
	calc.AverageEntryPriceFP = 0
	p.data[mID][tID] = calc
	p.mu.Unlock()
}

func updatePosition(p *Position, e events.SettlePosition) {
	// if this settlePosition event has a margin event embedded, that means we're dealing
	// with a trader who was closed out...
	if margin, ok := e.Margin(); ok {
		p.RealisedPNL += p.UnrealisedPNL
		p.RealisedPNLFP += p.UnrealisedPNLFP
		p.OpenVolume = 0
		p.UnrealisedPNL = 0
		p.AverageEntryPrice = 0
		// realised P&L includes whatever we had in margin account at this point
		p.RealisedPNL -= int64(margin)
		p.RealisedPNLFP -= float64(margin)
		// @TODO average entry price shouldn't be affected(?)
		// the volume now is zero, though, so we'll end up moving this position to storage
		p.UnrealisedPNLFP = 0
		p.AverageEntryPriceFP = 0
		return
	}
	for _, t := range e.Trades() {
		openedVolume, closedVolume := calculateOpenClosedVolume(p.OpenVolume, t.Size())
		_ = closeV(p, closedVolume, t.Price())
		openV(p, openedVolume, t.Price())
		p.AverageEntryPrice = uint64(math.Round(p.AverageEntryPriceFP))
		p.RealisedPNL = int64(math.Round(p.RealisedPNLFP))
	}
	mtm(p, e.Price())
	p.UnrealisedPNL = int64(math.Round(p.UnrealisedPNLFP))
}

type Position struct {
	types.Position
	AverageEntryPriceFP float64
	RealisedPNLFP       float64
	UnrealisedPNLFP     float64

	// what the party lost because of loss socialization
	loss float64
	// what a party was missing which triggered loss socialization
	adjustment float64
}

func speToProto(e SPE) Position {
	return Position{
		Position: types.Position{
			MarketID: e.MarketID(),
			PartyID:  e.Party(),
		},
		AverageEntryPriceFP: 0,
		RealisedPNLFP:       0,
		UnrealisedPNLFP:     0,
	}
}

func evtToProto(e events.SettlePosition) Position {
	p := Position{
		Position: types.Position{
			MarketID: e.MarketID(),
			PartyID:  e.Party(),
		},
		AverageEntryPriceFP: 0,
		RealisedPNLFP:       0,
		UnrealisedPNLFP:     0,
	}
	// NOTE: We don't call this here because the call is made in updateEvt for all positions
	// we don't want to add the same data twice!
	// updatePosition(&p, e)
	return p
}

func absUint64(v int64) uint64 {
	if v < 0 {
		v *= -1
	}
	return uint64(v)
}

func (p *Positions) Types() []events.Type {
	return []events.Type{
		events.SettlePositionEvent,
		events.SettleDistressedEvent,
		events.LossSocializationEvent,
	}
}
