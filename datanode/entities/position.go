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

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

type positionSettlement interface {
	Price() *num.Uint
	PositionFactor() num.Decimal
	Trades() []events.TradeSettlement
	TxHash() string
}

type lossSocialization interface {
	Amount() *num.Int
	TxHash() string
}

type settleDistressed interface {
	Margin() *num.Uint
	TxHash() string
}

type settleMarket interface {
	SettledPrice() *num.Uint
	PositionFactor() num.Decimal
	TxHash() string
}

type Position struct {
	MarketID                MarketID
	PartyID                 PartyID
	OpenVolume              int64
	RealisedPnl             decimal.Decimal
	UnrealisedPnl           decimal.Decimal
	AverageEntryPrice       decimal.Decimal
	AverageEntryMarketPrice decimal.Decimal
	Loss                    decimal.Decimal // what the party lost because of loss socialization
	Adjustment              decimal.Decimal // what a party was missing which triggered loss socialization
	TxHash                  TxHash
	VegaTime                time.Time
	// keep track of trades that haven't been settled as separate fields
	// these will be zeroed out once we process settlement events
	PendingOpenVolume              int64
	PendingRealisedPnl             decimal.Decimal
	PendingUnrealisedPnl           decimal.Decimal
	PendingAverageEntryPrice       decimal.Decimal
	PendingAverageEntryMarketPrice decimal.Decimal
	LossSocialisationAmount        decimal.Decimal
	DistressedStatus               PositionStatus
	TakerFeesPaid                  *num.Uint
	MakerFeesReceived              *num.Uint
	FeesPaid                       *num.Uint // infra fees and the like
	TakerFeesPaidSince             *num.Uint
	MakerFeesReceivedSince         *num.Uint
	FeesPaidSince                  *num.Uint
	FundingPaymentAmount           *num.Int
	FundingPaymentAmountSince      *num.Int
}

func NewEmptyPosition(marketID MarketID, partyID PartyID) Position {
	return Position{
		MarketID:                       marketID,
		PartyID:                        partyID,
		OpenVolume:                     0,
		RealisedPnl:                    num.DecimalZero(),
		UnrealisedPnl:                  num.DecimalZero(),
		AverageEntryPrice:              num.DecimalZero(),
		AverageEntryMarketPrice:        num.DecimalZero(),
		Loss:                           num.DecimalZero(),
		Adjustment:                     num.DecimalZero(),
		PendingOpenVolume:              0,
		PendingRealisedPnl:             num.DecimalZero(),
		PendingUnrealisedPnl:           num.DecimalZero(),
		PendingAverageEntryPrice:       num.DecimalZero(),
		PendingAverageEntryMarketPrice: num.DecimalZero(),
		LossSocialisationAmount:        num.DecimalZero(),
		DistressedStatus:               PositionStatusUnspecified,
		TakerFeesPaid:                  num.UintZero(),
		MakerFeesReceived:              num.UintZero(),
		FeesPaid:                       num.UintZero(),
		TakerFeesPaidSince:             num.UintZero(),
		MakerFeesReceivedSince:         num.UintZero(),
		FeesPaidSince:                  num.UintZero(),
		FundingPaymentAmount:           num.IntZero(),
		FundingPaymentAmountSince:      num.IntZero(),
	}
}

func (p *Position) updateWithBadTrade(trade vega.Trade, seller bool, pf num.Decimal) {
	size := int64(trade.Size)
	if seller {
		size *= -1
	}
	// update the open volume (not pending) directly, otherwise the settle position event resets the network position.
	price, _ := num.UintFromString(trade.AssetPrice, 10)
	mPrice, _ := num.UintFromString(trade.Price, 10)

	openedVolume, closedVolume := CalculateOpenClosedVolume(p.OpenVolume, size)
	realisedPnlDelta := num.DecimalFromUint(price).Sub(p.AverageEntryPrice).Mul(num.DecimalFromInt64(closedVolume)).Div(pf)
	p.RealisedPnl = p.RealisedPnl.Add(realisedPnlDelta)
	p.OpenVolume -= closedVolume

	p.AverageEntryPrice = updateVWAP(p.AverageEntryPrice, p.OpenVolume, openedVolume, price)
	p.AverageEntryMarketPrice = updateVWAP(p.AverageEntryMarketPrice, p.OpenVolume, openedVolume, mPrice)
	p.OpenVolume += openedVolume
	// no MTM - this isn't a settlement event, we're just adding the trade adding distressed volume to network
	// for the same reason, no syncPending call.
}

func (p *Position) UpdateWithTrade(trade vega.Trade, seller bool, pf num.Decimal) {
	// we have to ensure that we know the price/position factor
	size := int64(trade.Size)
	if seller {
		size *= -1
	}
	// add fees paid/received
	fees := getFeeAmountsForSide(&trade, seller)
	p.MakerFeesReceived.AddSum(fees.maker)
	p.TakerFeesPaid.AddSum(fees.taker)
	p.FeesPaid.AddSum(fees.other)
	// check if we should reset the "since" fields for fees
	since := p.PendingOpenVolume == 0
	// close out trade doesn't require the MTM calculation to be performed
	// the distressed trader will be handled through a settle distressed event, the network
	// open volume should just be updated, the average entry price is unchanged.
	assetPrice, _ := num.DecimalFromString(trade.AssetPrice)
	marketPrice, _ := num.DecimalFromString(trade.Price)

	// Scale the trade to the correct size
	opened, closed := CalculateOpenClosedVolume(p.PendingOpenVolume, size)
	realisedPnlDelta := assetPrice.Sub(p.PendingAverageEntryPrice).Mul(num.DecimalFromInt64(closed)).Div(pf)
	p.PendingRealisedPnl = p.PendingRealisedPnl.Add(realisedPnlDelta)
	// did we start with a positive/negative position?
	pos := p.PendingOpenVolume > 0
	p.PendingOpenVolume -= closed

	marketPriceUint, _ := num.UintFromDecimal(marketPrice)
	assetPriceUint, _ := num.UintFromDecimal(assetPrice)

	p.PendingAverageEntryPrice = updateVWAP(p.PendingAverageEntryPrice, p.PendingOpenVolume, opened, assetPriceUint)
	p.PendingAverageEntryMarketPrice = updateVWAP(p.PendingAverageEntryMarketPrice, p.PendingOpenVolume, opened, marketPriceUint)
	p.PendingOpenVolume += opened
	// either the position is no longer 0, or the position has flipped sides (and is non-zero)
	if since || (pos != (p.PendingOpenVolume > 0) && p.PendingOpenVolume != 0) {
		p.MakerFeesReceivedSince = num.UintZero()
		p.TakerFeesPaidSince = num.UintZero()
		p.FeesPaidSince = num.UintZero()
	}
	if p.PendingOpenVolume != 0 {
		// running total of fees paid since get incremented
		p.MakerFeesReceivedSince.AddSum(fees.maker)
		p.TakerFeesPaidSince.AddSum(fees.taker)
		p.FeesPaidSince.AddSum(fees.other)
	}
	p.pendingMTM(assetPrice, pf)
	if trade.Type == types.TradeTypeNetworkCloseOutBad {
		p.updateWithBadTrade(trade, seller, pf)
	} else if p.DistressedStatus == PositionStatusClosedOut {
		// Not a closeout trade, but the position is currently still marked as distressed.
		// This indicates the party was closed out previously, but has topped up and opened a new position.
		p.DistressedStatus = PositionStatusUnspecified
	}
}

func (p *Position) ApplyFundingPayment(amount *num.Int) {
	p.FundingPaymentAmount.Add(amount)
	p.FundingPaymentAmountSince.Add(amount)
	// da := num.DecimalFromInt(amount)
	// p.PendingRealisedPnl = p.PendingRealisedPnl.Add(da)
	// p.RealisedPnl = p.RealisedPnl.Add(da)
}

func (p *Position) UpdateOrdersClosed() {
	p.DistressedStatus = PositionStatusOrdersClosed
}

func (p *Position) ToggleDistressedStatus() {
	// if currently marked as distressed -> mark as safe
	if p.DistressedStatus == PositionStatusDistressed {
		p.DistressedStatus = PositionStatusUnspecified
		return
	}
	// was safe, is now distressed
	p.DistressedStatus = PositionStatusDistressed
}

func (p *Position) UpdateWithPositionSettlement(e positionSettlement) {
	pf := e.PositionFactor()
	resetFP := false
	for _, t := range e.Trades() {
		if p.OpenVolume == 0 {
			resetFP = true
		}
		openedVolume, closedVolume := CalculateOpenClosedVolume(p.OpenVolume, t.Size())
		// Deal with any volume we have closed
		realisedPnlDelta := num.DecimalFromUint(t.Price()).Sub(p.AverageEntryPrice).Mul(num.DecimalFromInt64(closedVolume)).Div(pf)
		p.RealisedPnl = p.RealisedPnl.Add(realisedPnlDelta)
		pos := p.OpenVolume > 0
		p.OpenVolume -= closedVolume

		// Then with any we have opened
		p.AverageEntryPrice = updateVWAP(p.AverageEntryPrice, p.OpenVolume, openedVolume, t.Price())
		p.AverageEntryMarketPrice = updateVWAP(p.AverageEntryMarketPrice, p.OpenVolume, openedVolume, t.MarketPrice())
		p.OpenVolume += openedVolume
		// check if position flipped
		if !resetFP && (pos != (p.OpenVolume > 0) && p.OpenVolume != 0) {
			resetFP = true
		}
	}
	if resetFP {
		p.FundingPaymentAmountSince = num.IntZero()
	}
	p.mtm(e.Price(), pf)
	p.TxHash = TxHash(e.TxHash())
	p.syncPending()
}

func (p *Position) syncPending() {
	// update pending fields to match current ones
	p.PendingOpenVolume = p.OpenVolume
	p.PendingRealisedPnl = p.RealisedPnl
	p.PendingUnrealisedPnl = p.UnrealisedPnl
	p.PendingAverageEntryPrice = p.AverageEntryPrice
	p.PendingAverageEntryMarketPrice = p.AverageEntryMarketPrice
}

func (p *Position) UpdateWithLossSocialization(e lossSocialization) {
	amountLoss := num.DecimalFromInt(e.Amount())

	if amountLoss.IsNegative() {
		p.Loss = p.Loss.Add(amountLoss)
		p.LossSocialisationAmount = p.LossSocialisationAmount.Sub(amountLoss)
	} else {
		p.Adjustment = p.Adjustment.Add(amountLoss)
		p.LossSocialisationAmount = p.LossSocialisationAmount.Add(amountLoss)
	}

	p.RealisedPnl = p.RealisedPnl.Add(amountLoss)
	p.TxHash = TxHash(e.TxHash())
	p.syncPending()
}

func (p *Position) UpdateWithSettleDistressed(e settleDistressed) {
	margin := num.DecimalFromUint(e.Margin())
	p.RealisedPnl = p.RealisedPnl.Add(p.UnrealisedPnl)
	p.RealisedPnl = p.RealisedPnl.Sub(margin) // realised P&L includes whatever we had in margin account at this point
	p.UnrealisedPnl = num.DecimalZero()
	p.AverageEntryPrice = num.DecimalZero() // @TODO average entry price shouldn't be affected(?)
	p.AverageEntryPrice = num.DecimalZero()
	p.OpenVolume = 0
	p.TxHash = TxHash(e.TxHash())
	p.DistressedStatus = PositionStatusClosedOut
	p.FundingPaymentAmountSince = num.IntZero()
	p.FeesPaidSince = num.UintZero()
	p.MakerFeesReceivedSince = num.UintZero()
	p.TakerFeesPaidSince = num.UintZero()
	p.syncPending()
}

func (p *Position) UpdateWithSettleMarket(e settleMarket) {
	markPriceDec := num.DecimalFromUint(e.SettledPrice())
	openVolumeDec := num.DecimalFromInt64(p.OpenVolume)

	unrealisedPnl := openVolumeDec.Mul(markPriceDec.Sub(p.AverageEntryPrice)).Div(e.PositionFactor())
	p.RealisedPnl = p.RealisedPnl.Add(unrealisedPnl)
	p.UnrealisedPnl = num.DecimalZero()
	p.OpenVolume = 0
	p.TxHash = TxHash(e.TxHash())
	p.syncPending()
}

func (p Position) ToProto() *vega.Position {
	var timestamp int64
	if !p.VegaTime.IsZero() {
		timestamp = p.VegaTime.UnixNano()
	}
	// we use the pending values when converting to protos
	// so trades are reflected as accurately as possible
	return &vega.Position{
		MarketId:                  p.MarketID.String(),
		PartyId:                   p.PartyID.String(),
		OpenVolume:                p.PendingOpenVolume,
		RealisedPnl:               p.PendingRealisedPnl.Round(0).String(),
		UnrealisedPnl:             p.PendingUnrealisedPnl.Round(0).String(),
		AverageEntryPrice:         p.PendingAverageEntryMarketPrice.Round(0).String(),
		UpdatedAt:                 timestamp,
		LossSocialisationAmount:   p.LossSocialisationAmount.Round(0).String(),
		PositionStatus:            vega.PositionStatus(p.DistressedStatus),
		TakerFeesPaid:             p.TakerFeesPaid.String(),
		MakerFeesReceived:         p.MakerFeesReceived.String(),
		FeesPaid:                  p.FeesPaid.String(),
		TakerFeesPaidSince:        p.TakerFeesPaidSince.String(),
		MakerFeesReceivedSince:    p.MakerFeesReceivedSince.String(),
		FeesPaidSince:             p.FeesPaidSince.String(),
		FundingPaymentAmount:      p.FundingPaymentAmount.String(),
		FundingPaymentAmountSince: p.FundingPaymentAmountSince.String(),
	}
}

func (p Position) ToProtoEdge(_ ...any) (*v2.PositionEdge, error) {
	return &v2.PositionEdge{
		Node:   p.ToProto(),
		Cursor: p.Cursor().Encode(),
	}, nil
}

func (p *Position) AverageEntryPriceUint() *num.Uint {
	uint, overflow := num.UintFromDecimal(p.AverageEntryPrice)
	if overflow {
		panic("couldn't convert average entry price from decimal to uint")
	}
	return uint
}

func (p *Position) mtm(markPrice *num.Uint, positionFactor num.Decimal) {
	if p.OpenVolume == 0 {
		p.UnrealisedPnl = num.DecimalZero()
		return
	}
	markPriceDec := num.DecimalFromUint(markPrice)
	openVolumeDec := num.DecimalFromInt64(p.OpenVolume)

	p.UnrealisedPnl = openVolumeDec.Mul(markPriceDec.Sub(p.AverageEntryPrice)).Div(positionFactor)
}

func (p *Position) pendingMTM(price, sf num.Decimal) {
	if p.PendingOpenVolume == 0 {
		p.PendingUnrealisedPnl = num.DecimalZero()
		return
	}

	vol := num.DecimalFromInt64(p.PendingOpenVolume)
	p.PendingUnrealisedPnl = vol.Mul(price.Sub(p.PendingAverageEntryPrice)).Div(sf)
}

func CalculateOpenClosedVolume(currentOpenVolume, tradedVolume int64) (int64, int64) {
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

func absUint64(v int64) uint64 {
	if v < 0 {
		v *= -1
	}
	return uint64(v)
}

func updateVWAP(vwap num.Decimal, volume int64, addVolume int64, addPrice *num.Uint) num.Decimal {
	if volume+addVolume == 0 {
		return num.DecimalZero()
	}

	volumeDec := num.DecimalFromInt64(volume)
	addVolumeDec := num.DecimalFromInt64(addVolume)
	addPriceDec := num.DecimalFromUint(addPrice)

	return vwap.Mul(volumeDec).Add(addPriceDec.Mul(addVolumeDec)).Div(volumeDec.Add(addVolumeDec))
}

type PositionKey struct {
	MarketID MarketID
	PartyID  PartyID
	VegaTime time.Time
}

func (p Position) Cursor() *Cursor {
	pc := PositionCursor{
		MarketID: p.MarketID,
		PartyID:  p.PartyID,
		VegaTime: p.VegaTime,
	}

	return NewCursor(pc.String())
}

func (p Position) Key() PositionKey {
	return PositionKey{p.MarketID, p.PartyID, p.VegaTime}
}

var PositionColumns = []string{
	"market_id", "party_id", "open_volume", "realised_pnl", "unrealised_pnl",
	"average_entry_price", "average_entry_market_price", "loss", "adjustment", "tx_hash", "vega_time", "pending_open_volume",
	"pending_realised_pnl", "pending_unrealised_pnl", "pending_average_entry_price", "pending_average_entry_market_price",
	"loss_socialisation_amount", "distressed_status", "taker_fees_paid", "maker_fees_received", "fees_paid",
	"taker_fees_paid_since", "maker_fees_received_since", "fees_paid_since", "funding_payment_amount", "funding_payment_amount_since",
}

func (p Position) ToRow() []interface{} {
	return []interface{}{
		p.MarketID, p.PartyID, p.OpenVolume, p.RealisedPnl, p.UnrealisedPnl,
		p.AverageEntryPrice, p.AverageEntryMarketPrice, p.Loss, p.Adjustment, p.TxHash, p.VegaTime, p.PendingOpenVolume,
		p.PendingRealisedPnl, p.PendingUnrealisedPnl, p.PendingAverageEntryPrice, p.PendingAverageEntryMarketPrice,
		p.LossSocialisationAmount, p.DistressedStatus, p.TakerFeesPaid, p.MakerFeesReceived, p.FeesPaid,
		p.TakerFeesPaidSince, p.MakerFeesReceivedSince, p.FeesPaidSince, p.FundingPaymentAmount, p.FundingPaymentAmountSince,
	}
}

func (p Position) Equal(q Position) bool {
	return p.MarketID == q.MarketID &&
		p.PartyID == q.PartyID &&
		p.OpenVolume == q.OpenVolume &&
		p.RealisedPnl.Equal(q.RealisedPnl) &&
		p.UnrealisedPnl.Equal(q.UnrealisedPnl) &&
		p.AverageEntryPrice.Equal(q.AverageEntryPrice) &&
		p.AverageEntryMarketPrice.Equal(q.AverageEntryMarketPrice) &&
		p.Loss.Equal(q.Loss) &&
		p.Adjustment.Equal(q.Adjustment) &&
		p.TxHash == q.TxHash &&
		p.VegaTime.Equal(q.VegaTime) &&
		p.PendingOpenVolume == q.PendingOpenVolume &&
		p.PendingAverageEntryPrice.Equal(q.PendingAverageEntryPrice) &&
		p.PendingAverageEntryMarketPrice.Equal(q.PendingAverageEntryMarketPrice) &&
		p.PendingRealisedPnl.Equal(q.PendingRealisedPnl) &&
		p.PendingUnrealisedPnl.Equal(q.PendingUnrealisedPnl)
	// p.PendingUnrealisedPnl.Equal(q.PendingUnrealisedPnl) &&
	// loss socialisation amount doesn't seem to work currently
	// p.LossSocialisationAmount.Equal(q.LossSocialisationAmount) &&
	// p.DistressedStatus == q.DistressedStatus
}

type PositionCursor struct {
	VegaTime time.Time `json:"vega_time"`
	PartyID  PartyID   `json:"party_id"`
	MarketID MarketID  `json:"market_id"`
}

func (rc PositionCursor) String() string {
	bs, err := json.Marshal(rc)
	if err != nil {
		// This should never happen.
		panic(fmt.Errorf("could not marshal order cursor: %w", err))
	}
	return string(bs)
}

func (rc *PositionCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), rc)
}
