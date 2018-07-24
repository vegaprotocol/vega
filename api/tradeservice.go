package api

import (
	"context"
	"errors"
	"time"

	"vega/core"
	"vega/datastore"
	"vega/proto"
)

type TradeService interface {
	Init(app *core.Vega, tradeStore datastore.TradeStore)
	GetByMarket(ctx context.Context, market string, limit uint64) (trades []*msg.Trade, err error)
	GetByParty(ctx context.Context, party string, limit uint64) (trades []*msg.Trade, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (trade *msg.Trade, err error)
	GetByPartyAndId(ctx context.Context, party string, id string) (trade *msg.Trade, err error)
	GetCandles(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error)
	GetPositionsByParty(ctx context.Context, party string) (positions []*msg.MarketPosition, err error)
}

type tradeService struct {
	app        *core.Vega
	tradeStore datastore.TradeStore
}

func NewTradeService() TradeService {
	return &tradeService{}
}

func (t *tradeService) Init(app *core.Vega, tradeStore datastore.TradeStore) {
	t.app = app
	t.tradeStore = tradeStore
}

func (t *tradeService) GetByMarket(ctx context.Context, market string, limit uint64) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByMarket(market, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]*msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetByParty(ctx context.Context, party string, limit uint64) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByParty(party, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]*msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetByMarketAndId(ctx context.Context, market string, id string) (trade *msg.Trade, err error) {
	tr, err := t.tradeStore.GetByMarketAndId(market, id)
	if err != nil {
		return &msg.Trade{}, err
	}
	return tr.ToProtoMessage(), err
}

func (t *tradeService) GetByPartyAndId(ctx context.Context, party string, id string) (trade *msg.Trade, err error) {
	tr, err := t.tradeStore.GetByPartyAndId(party, id)
	if err != nil {
		return &msg.Trade{}, err
	}
	return tr.ToProtoMessage(), err
}

func (t *tradeService) GetCandles(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error) {
	// compare time and translate it into timestamps
	appCurrentTime := t.app.GetTime()

	delta := appCurrentTime.Sub(since)
	deltaInSeconds := int64(delta.Seconds())
	if deltaInSeconds < 0 {
		return msg.Candles{}, errors.New("INVALID_REQUEST")
	}

	sinceBlock := t.app.GetAbciHeight() - deltaInSeconds
	if sinceBlock < 0 {
		sinceBlock = 0
	}

	c, err := t.tradeStore.GetCandles(market, uint64(sinceBlock), uint64(t.app.GetAbciHeight()), interval)
	if err != nil {
		return msg.Candles{}, err
	}

	aggregationStartTime := appCurrentTime.Add(-delta)
	for i, candle := range c.Candles {
		candleDuration := time.Duration(i*int(interval)) * time.Second
		candle.Date = aggregationStartTime.Add(candleDuration).Format(time.RFC3339)
	}

	return c, nil
}

func (t *tradeService) GetPositionsByParty(ctx context.Context, party string) (positions []*msg.MarketPosition, err error) {
	mapOfMarketPositions := t.tradeStore.GetPositionsByParty(party)
	for _, marketPositions := range mapOfMarketPositions {
		positions = append(positions, marketPositions)
	}
	return positions, nil
}

//func (t *tradeService) GetPositionsByParty(ctx context.Context, party string) (positions []*msg.Position, err error) {
//	mapOfNetPositions := t.tradeStore.GetNetPositionsByParty(party)
//
//	var (
//		absoluteExposure int64
//		PNL              int64
//		direction        string
//		profitOrLoss     string
//	)
//
//	for marketName, exposure := range mapOfNetPositions {
//		// get last traded price for this market
//		currentPrice, _ := t.tradeStore.GetCurrentMarketPrice(marketName)
//		fmt.Printf("current price for market %s is %d\n", marketName, currentPrice)
//
//		// calculate absolute value of the exposure on that market
//		absoluteExposure = int64(math.Abs(float64(exposure.Position)))
//
//		// calculate profit and loss which is currentPrice * Abs(volume of current exposure) - absoluteExposure on that market
//		//volume weighted price
//		PNL = int64(int64(currentPrice)*int64(math.Abs(float64(exposure.Volume))) - absoluteExposure)
//
//		// exposure.Position is negative for shorts. Check sign to get direction of position on that market
//		direction = getDirection(exposure.Position)
//
//		// verify whether direction of your position is aligned with your PNL on that market
//		profitOrLoss = getPNLResult(direction, PNL)
//
//		// this calculates position per market. Append to list of positions.
//		newPosition := &msg.Position{
//			Market:    marketName,
//			Direction: direction,
//			Exposure:  absoluteExposure,
//			PNL:       PNL,
//			PNLResult: profitOrLoss,
//		}
//		positions = append(positions, newPosition)
//	}
//
//	return positions, nil
//}
//
//func getDirection(val int64) string {
//	if val > 0 {
//		return LongPosition
//	}
//	if val < 0 {
//		return ShortPosition
//	}
//	return Net
//}
//
//func getPNLResult(direction string, PNL int64) string {
//	if PNL == 0 {
//		return Net
//	}
//
//	// if trader shorts and PNL at current market price is negative that means he is making money
//	if direction == ShortPosition && PNL < 0 {
//		return Profit
//	} else {
//		return Loss
//	}
//
//	// if trader is long and PNL at current market price is positive that means he is making money
//	if direction == LongPosition && PNL > 0 {
//		return Profit
//	} else {
//		return Loss
//	}
//	return ""
//}
