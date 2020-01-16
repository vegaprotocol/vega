package buffer_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/events"

	"github.com/stretchr/testify/assert"
)

type settleTst struct {
	*buffer.Settlement
	ctx   context.Context
	cfunc context.CancelFunc
}

type settlePosEvt struct {
	market          string
	party           string
	size, buy, sell int64
	price           uint64
	trades          []events.TradeSettlement
}

func (s *settlePosEvt) Party() string {
	return s.party
}

func (s *settlePosEvt) Size() int64 {
	return s.size
}

func (s *settlePosEvt) Buy() int64 {
	return s.buy
}

func (s *settlePosEvt) Sell() int64 {
	return s.sell
}

func (s *settlePosEvt) Price() uint64 {
	return s.price
}

func (s *settlePosEvt) MarketID() string {
	return s.market
}

func (s *settlePosEvt) Trades() []events.TradeSettlement {
	return s.trades
}

func TestNewSettleCh(t *testing.T) {
	buf := getSettleTst()
	buf.cfunc()
}

func TestGetSettleSub(t *testing.T) {
	buf := getSettleTst()
	sub1 := buf.Subscribe(1)
	sub2 := buf.Subscribe(1)
	assert.NotNil(t, sub1)
	assert.NotNil(t, sub2)
	buf.Unsubscribe(sub1)
	// the first should error, the second sub shouldn't
	assert.Error(t, sub1.Err())
	assert.NoError(t, sub2.Err())
	_, ok := <-sub1.Recv()
	assert.False(t, ok)
	sub1 = buf.Subscribe(1)
	buf.cfunc()
	// ctx is cancelled, sub channels should be closed
	<-sub1.Done()
	<-sub2.Done()
	_, ok = <-sub1.Recv()
	assert.False(t, ok)
	_, ok = <-sub2.Recv()
	assert.False(t, ok)
}

func TestSettleBufFlush(t *testing.T) {
	buf := getSettleTst()
	sub := buf.Subscribe(1)
	mkt, party := "marketID", "party1"
	pos := &settlePosEvt{
		market: mkt,
		party:  party,
	}
	buf.Add([]events.SettlePosition{pos})
	buf.Flush()
	data := <-sub.Recv()
	assert.Equal(t, 1, len(data))
	assert.Equal(t, pos, data[0])
	assert.NoError(t, sub.Err())
	buf.cfunc() // close down everything
	assert.NotNil(t, <-sub.Done())
	assert.Error(t, sub.Err())
}

func TestSettleBufLastOnly(t *testing.T) {
	buf := getSettleTst()
	sub := buf.Subscribe(1)
	party, party2, mkt := "trader-1", "trader-2", "test-market"
	data := []events.SettlePosition{
		&settlePosEvt{
			market: mkt,
			party:  party,
		},
		&settlePosEvt{
			market: mkt,
			party:  party,
		},
		&settlePosEvt{
			market: mkt,
			party:  party2,
		},
	}
	// keep pushing accounts onto buffer, only the last should remain
	buf.Add(data)
	buf.Flush()
	received := <-sub.Recv()
	assert.Equal(t, 2, len(received))
	// we expect the ID to be set to empty string by the buffer
	assert.Equal(t, data[1], received[0])
	assert.Equal(t, data[2], received[1])
	assert.NoError(t, sub.Err())
	buf.cfunc() // close down everything
	assert.NotNil(t, <-sub.Done())
	assert.Error(t, sub.Err())
}

func getSettleTst() *settleTst {
	ctx, cfunc := context.WithCancel(context.Background())
	return &settleTst{
		Settlement: buffer.NewSettlement(ctx),
		ctx:        ctx,
		cfunc:      cfunc,
	}
}
