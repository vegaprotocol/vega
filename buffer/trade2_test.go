package buffer_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/buffer"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

type tradeBufTst struct {
	*buffer.TradeCh
	ctx   context.Context
	cfunc context.CancelFunc
}

type tstSub struct {
	key int
	ch  <-chan []types.Trade
}

// basic mechanics, the buffers are set up correctly
func TestTradeBufCtx(t *testing.T) {
	buf := getTradeBuf()
	buf.cfunc()
}

func TestTradeBufSubUnsub(t *testing.T) {
	buf := getTradeBuf()
	sub1 := buf.Subscribe(1)
	sub2 := buf.Subscribe(1)
	assert.NotNil(t, sub1.Recv())
	assert.NotNil(t, sub2.Recv())
	buf.Unsubscribe(sub1)
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

func TestTradeBufFlush(t *testing.T) {
	buf := getTradeBuf()
	sub := buf.Subscribe(1)
	trade := types.Trade{
		Id:       "trade-1",
		MarketID: "tst-market",
	}
	buf.Add(trade)
	buf.Flush()
	subTrade := <-sub.Recv()
	assert.Equal(t, 1, len(subTrade))
	assert.Equal(t, trade, subTrade[0])
	assert.NoError(t, sub.Err())
	buf.cfunc() // close down everything
	assert.NotNil(t, <-sub.Done())
	assert.Error(t, sub.Err())
}

func getTradeBuf() *tradeBufTst {
	ctx, cfunc := context.WithCancel(context.Background())
	return &tradeBufTst{
		TradeCh: buffer.NewTradeCh(ctx),
		cfunc:   cfunc,
	}
}
