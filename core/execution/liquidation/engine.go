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

package liquidation

import (
	"context"
	"fmt"
	"math"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/execution/liquidation Book,MarketLiquidity,IDGen,Positions,Settlement

type Book interface {
	GetVolumeAtPrice(price *num.Uint, side types.Side) uint64
}

type MarketLiquidity interface {
	ValidOrdersPriceRange() (*num.Uint, *num.Uint, error)
}

type IDGen interface {
	NextID() string
}

type Positions interface {
	RegisterOrder(ctx context.Context, order *types.Order) *positions.MarketPosition
	Update(ctx context.Context, trade *types.Trade, passiveOrder, aggressiveOrder *types.Order) []events.MarketPosition
}

type Settlement interface {
	AddTrade(trade *types.Trade)
}

type Engine struct {
	// settings, orderbook, network pos data
	log      *logging.Logger
	cfg      *types.LiquidationStrategy
	broker   common.Broker
	mID      string
	pos      *Pos
	book     Book
	as       common.AuctionState
	nextStep time.Time
	tSvc     common.TimeService
	ml       MarketLiquidity
	position Positions
	settle   Settlement
	stopped  bool
}

// protocol upgrade - default values for existing markets/proposals.
var (
	defaultStrat = &types.LiquidationStrategy{
		DisposalTimeStep:    10 * time.Second,
		DisposalFraction:    num.DecimalFromFloat(0.1),
		FullDisposalSize:    20,
		MaxFractionConsumed: num.DecimalFromFloat(0.05),
	}

	// this comes closest to the existing behaviour (trying to close the network position in full in one go).
	legacyStrat = &types.LiquidationStrategy{
		DisposalTimeStep:    0 * time.Second,
		DisposalFraction:    num.DecimalOne(),
		FullDisposalSize:    math.MaxUint64,
		MaxFractionConsumed: num.DecimalOne(),
	}
)

// GetDefaultStrat is exporeted, expected to be used to update existing proposals on protocol upgrade
// once that's happened, this code can be removed.
func GetDefaultStrat() *types.LiquidationStrategy {
	return defaultStrat.DeepClone()
}

// GetLegacyStrat is exported, same as defaul. This can be used for protocol upgrade
// it most closely resebles the old behaviour (network attempts to close out fully, in one go)
// this can be removed once protocol upgrade has completed.
func GetLegacyStrat() *types.LiquidationStrategy {
	return legacyStrat.DeepClone()
}

func New(log *logging.Logger, cfg *types.LiquidationStrategy, mktID string, broker common.Broker, book Book, as common.AuctionState, tSvc common.TimeService, ml MarketLiquidity, pe Positions, se Settlement) *Engine {
	// NOTE: This can be removed after protocol upgrade
	if cfg == nil {
		cfg = legacyStrat.DeepClone()
	}
	return &Engine{
		log:      log,
		cfg:      cfg,
		broker:   broker,
		mID:      mktID,
		book:     book,
		as:       as,
		tSvc:     tSvc,
		ml:       ml,
		position: pe,
		settle:   se,
		pos:      &Pos{},
	}
}

func (e *Engine) Update(cfg *types.LiquidationStrategy) {
	if !e.nextStep.IsZero() {
		since := e.nextStep.Add(-e.cfg.DisposalTimeStep) // work out when the network position was last updated
		e.nextStep = since.Add(cfg.DisposalTimeStep)
	}
	// now update the config
	e.cfg = cfg
}

func (e *Engine) OnTick(ctx context.Context, now time.Time) (*types.Order, error) {
	if e.pos.open == 0 || e.as.InAuction() || e.nextStep.After(now) {
		return nil, nil
	}
	minP, maxP, err := e.ml.ValidOrdersPriceRange()
	if err != nil {
		return nil, err
	}
	vol := e.pos.open
	bookSide := types.SideBuy
	side := types.SideSell
	bound := minP
	price := minP
	if vol < 0 {
		vol *= -1
		side, bookSide = bookSide, side
		price, bound = maxP, maxP
	}
	size := uint64(vol)
	if size > e.cfg.FullDisposalSize {
		// absolute size of network position * disposal fraction -> rounded
		size = uint64(num.DecimalFromFloat(float64(size)).Mul(e.cfg.DisposalFraction).Round(0).IntPart())
	}
	available := e.book.GetVolumeAtPrice(bound, bookSide)
	if available == 0 {
		return nil, nil
	}
	// round up, avoid a value like 0.1 to be floored, favour closing out a position of 1 at least
	maxCons := uint64(num.DecimalFromFloat(float64(available)).Mul(e.cfg.MaxFractionConsumed).Ceil().IntPart())
	if maxCons < size {
		size = maxCons
	}
	// get the block hash
	_, blockHash := vegacontext.TraceIDFromContext(ctx)
	idgen := idgeneration.New(blockHash + crypto.HashStrToHex("networkLS"+e.mID))
	// set time for next order, if the position ends up closed out, then that's fine
	// we'll remove this time when the position is updated
	if size == 0 {
		return nil, nil
	}
	e.nextStep = now.Add(e.cfg.DisposalTimeStep)
	// place order using size
	return &types.Order{
		ID:          idgen.NextID(),
		MarketID:    e.mID,
		Party:       types.NetworkParty,
		Side:        side,
		Price:       price,
		Size:        size,
		Remaining:   size,
		TimeInForce: types.OrderTimeInForceIOC,
		Type:        types.OrderTypeLimit,
		CreatedAt:   now.UnixNano(),
		Status:      types.OrderStatusActive,
		Reference:   "LS", // Liquidity sourcing
	}, nil
}

// ClearDistressedParties transfers the open positions to the network, returns the market position events and party ID's
// for the market to remove the parties from things like positions engine and collateral.
func (e *Engine) ClearDistressedParties(ctx context.Context, idgen IDGen, closed []events.Margin, mp, mmp *num.Uint) ([]events.MarketPosition, []string, []*types.Trade) {
	if len(closed) == 0 {
		return nil, nil, nil
	}
	// netork is most likely going to hold an actual position now, let's set up the time step when we attempt to dispose
	// of (some) of the volume
	if e.pos.open == 0 || e.nextStep.IsZero() {
		e.nextStep = e.tSvc.GetTimeNow().Add(e.cfg.DisposalTimeStep)
	}
	mps := make([]events.MarketPosition, 0, len(closed))
	parties := make([]string, 0, len(closed))
	// order events here
	orders := make([]events.Event, 0, len(closed)*2)
	// trade events here
	trades := make([]events.Event, 0, len(closed))
	netTrades := make([]*types.Trade, 0, len(closed))
	now := e.tSvc.GetTimeNow()
	for _, cp := range closed {
		e.pos.open += cp.Size()
		// get the orders and trades so we can send events to update the datanode
		o1, o2, t := e.getOrdersAndTrade(ctx, cp, idgen, now, mp, mmp)
		orders = append(orders, events.NewOrderEvent(ctx, o1), events.NewOrderEvent(ctx, o2))
		trades = append(trades, events.NewTradeEvent(ctx, *t))
		netTrades = append(netTrades, t)
		// add the confiscated balance to the fee pool that can be taken from the insurance pool to pay fees to
		// the good parties when the network closes itself out.
		mps = append(mps, cp)
		parties = append(parties, cp.Party())
	}
	// send order events
	e.broker.SendBatch(orders)
	// send trade events
	e.broker.SendBatch(trades)
	// the network has no (more) remaining open position -> no need for the e.nextStep to be set
	e.log.Info("network position after close-out", logging.Int64("network-position", e.pos.open))
	if e.pos.open == 0 {
		e.nextStep = time.Time{}
	}
	return mps, parties, netTrades
}

func (e *Engine) UpdateMarkPrice(mp *num.Uint) {
	e.pos.price = mp
}

func (e *Engine) GetNetworkPosition() events.MarketPosition {
	return e.pos
}

func (e *Engine) UpdateNetworkPosition(trades []*types.Trade) {
	sign := int64(1)
	if e.pos.open < 0 {
		sign *= -1
	}
	for _, t := range trades {
		delta := int64(t.Size) * sign
		e.pos.open -= delta
	}
	if e.pos.open == 0 {
		e.nextStep = time.Time{}
	} else if e.nextStep.IsZero() {
		e.nextStep = e.tSvc.GetTimeNow().Add(e.cfg.DisposalTimeStep)
	}
}

func (e *Engine) getOrdersAndTrade(ctx context.Context, pos events.Margin, idgen IDGen, now time.Time, price, dpPrice *num.Uint) (*types.Order, *types.Order, *types.Trade) {
	tSide, nSide := types.SideSell, types.SideBuy // one of them will have to sell
	s := pos.Size()
	size := uint64(s)
	if s < 0 {
		size = uint64(-s)
		// swap sides
		nSide, tSide = tSide, nSide
	}
	var buyID, sellID, buyParty, sellParty string
	order := types.Order{
		ID:            idgen.NextID(),
		MarketID:      e.mID,
		Status:        types.OrderStatusFilled,
		Party:         types.NetworkParty,
		Price:         price,
		OriginalPrice: dpPrice,
		CreatedAt:     now.UnixNano(),
		Reference:     "close-out distressed",
		TimeInForce:   types.OrderTimeInForceFOK, // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
		Type:          types.OrderTypeNetwork,
		Size:          size,
		Remaining:     size,
		Side:          nSide,
	}
	e.position.RegisterOrder(ctx, &order)
	order.Remaining = 0
	partyOrder := types.Order{
		ID:            idgen.NextID(),
		MarketID:      e.mID,
		Size:          size,
		Remaining:     size,
		Status:        types.OrderStatusFilled,
		Party:         pos.Party(),
		Side:          tSide, // assume sell, price is zero in that case anyway
		Price:         price, // average price
		OriginalPrice: dpPrice,
		CreatedAt:     now.UnixNano(),
		Reference:     fmt.Sprintf("distressed-%s", pos.Party()),
		TimeInForce:   types.OrderTimeInForceFOK, // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
		Type:          types.OrderTypeNetwork,
	}
	e.position.RegisterOrder(ctx, &partyOrder)
	partyOrder.Remaining = 0
	buyParty = order.Party
	sellParty = partyOrder.Party
	sellID = partyOrder.ID
	buyID = order.ID
	if tSide == types.SideBuy {
		sellID, buyID = buyID, sellID
		buyParty, sellParty = sellParty, buyParty
	}
	trade := types.Trade{
		ID:          idgen.NextID(),
		MarketID:    e.mID,
		Price:       price,
		MarketPrice: dpPrice,
		Size:        size,
		Aggressor:   order.Side, // we consider network to be aggressor
		BuyOrder:    buyID,
		SellOrder:   sellID,
		Buyer:       types.NetworkParty,
		Seller:      types.NetworkParty,
		Timestamp:   now.UnixNano(),
		Type:        types.TradeTypeNetworkCloseOutBad,
		SellerFee:   types.NewFee(),
		BuyerFee:    types.NewFee(),
	}
	// settlement engine should see this as a wash trade
	e.settle.AddTrade(&trade)
	trade.Buyer, trade.Seller = buyParty, sellParty
	// the for the rest of the core, this should not seem like a wash trade though...
	e.position.Update(ctx, &trade, &order, &partyOrder)
	return &order, &partyOrder, &trade
}

func (e *Engine) GetNextCloseoutTS() int64 {
	if e.nextStep.IsZero() {
		return 0
	}
	return e.nextStep.UnixNano()
}
