package api

import (
	"context"
	"errors"
	"math"
	"time"

	"vega/core"
	"vega/datastore"
	"vega/proto"
	"github.com/golang/go/src/pkg/fmt"
)

type TradeService interface {
	Init(app *core.Vega, tradeStore datastore.TradeStore)
	GetByMarket(ctx context.Context, market string, limit uint64) (trades []*msg.Trade, err error)
	GetByParty(ctx context.Context, party string, limit uint64) (trades []*msg.Trade, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (trade *msg.Trade, err error)
	GetByPartyAndId(ctx context.Context, party string, id string) (trade *msg.Trade, err error)
	GetCandles(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error)
	GetPositionsByParty(ctx context.Context, party string) (positions []*msg.Position, err error)
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

func (t *tradeService) GetPositionsByParty(ctx context.Context, party string) (positions []*msg.Position, err error) {
	mapOfNetPositions := t.tradeStore.GetNetPositionsByParty(party)
	var (
		profitOrLoss string
		exposure int64
		PNL int64
		direction string
	)

	for key, val := range mapOfNetPositions {
		currentPrice, _ := t.tradeStore.GetCurrentMarketPrice(key)
		fmt.Printf("current Price %d\n", currentPrice)
		exposure = int64(math.Abs(float64(val.Position)))
		PNL = int64(int64(currentPrice) * int64(math.Abs(float64(val.Volume))) - exposure)
		direction = getDirection(val.Position)
		profitOrLoss = getPNLResult(direction, PNL)

		newPosition := &msg.Position{
			Market: key,
			Direction: direction,
			Exposure: exposure,
			PNL: PNL,
			PNLResult: profitOrLoss,
		}
		positions = append(positions, newPosition)
	}

	return positions, nil
}

func getDirection(val int64) string{
	if val > 0 { return "LONG" }
	if val < 0 { return "SHORT"}
	return "NET"
}

func getPNLResult(direction string, PNL int64) string {
	if PNL == 0 {
		return ""
	}
	if direction == "SHORT" && PNL < 0 {
		return "PROFIT"
	} else {
		return "LOSS"
	}
	if direction == "LONG" && PNL > 0 {
		return "PROFIT"
	} else {
		return "LOSS"
	}
	return ""
}

//115 x 100 = za tyle to jest w tej chwili do opierdolenia
//moj exposure = (agg amount X Agg price) tyle mam tego gowna w tej chwili
//PNL = currentPrice - moj exposure
//if SHORT && PNL < 0 => PROFIT
//if LONG && PNL > 0 => PROFIT
