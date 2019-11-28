package buffer_test

import (
	"testing"

	"code.vegaprotocol.io/vega/buffer"
	"github.com/stretchr/testify/assert"
)

var (
	market = "test-mkt"
	chBuf  = 1
)

type mpEvt struct {
	party           string
	size, buy, sell int64
	price           uint64
}

func TestBufferUpdates(t *testing.T) {
	// buf := buffer.New(market, buffer.SetChannelBuffer(0))
	buf := buffer.New(market)
	data := []mpEvt{
		{
			party: "trader-1",
			size:  10,
			buy:   5,
			price: 1000,
		},
		{
			party: "trader-2",
			size:  -10,
			sell:  -2,
			price: 1000,
		},
	}
	for _, evt := range data {
		buf.Add(evt)
	}
	ch, id := buf.Register()
	assert.NotNil(t, ch)
	assert.Equal(t, 0, id)
	buf.Flush()
	update := <-ch
	assert.NotEmpty(t, update)
	assert.Contains(t, update, "trader-1")
	assert.Contains(t, update, "trader-2")
	t1 := update["trader-1"]
	assert.Equal(t, data[0].price, t1.AverageEntryPrice)
	assert.Equal(t, data[0].size, t1.RealisedVolume)

	// update trader 1 alone, see if average entry price and realised volume is updated
	buf.Add(mpEvt{
		party: "trader-1",
		size:  15,
		price: 2000,
	})
	buf.Flush()
	// read from channel again
	update = <-ch
	buf.Unregister(id)
	assert.NotEmpty(t, update)
	assert.Contains(t, update, "trader-1")
	t1 = update["trader-1"]
	assert.Equal(t, uint64(1333), t1.AverageEntryPrice)
}

func TestRegisterMultiple(t *testing.T) {
	chBuf := 5
	// just cover the optional channel size, too
	buf := buffer.New(market, buffer.SetChannelBuffer(chBuf))
	ch1, id1 := buf.Register()
	ch2, id2 := buf.Register()
	assert.NotEqual(t, id1, id2)
	assert.Equal(t, cap(ch1), cap(ch2))
	assert.Equal(t, chBuf, cap(ch1))
	buf.Unregister(id2)
	// register another channel, id should be the first available one
	_, id3 := buf.Register()
	assert.Equal(t, id2, id3)
	buf.Unregister(id1)
	buf.Unregister(id3)
}

func (m mpEvt) Size() int64 {
	return m.size
}

func (m mpEvt) Buy() int64 {
	return m.buy
}

func (m mpEvt) Sell() int64 {
	return m.sell
}

func (m mpEvt) Price() uint64 {
	return m.price
}

func (m mpEvt) ClearPotentials() {}

func (m mpEvt) Party() string {
	return m.party
}
