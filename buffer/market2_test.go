package buffer_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/buffer"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

type mktBufTst struct {
	*buffer.MarketCh
	ctx   context.Context
	cfunc context.CancelFunc
}

// basic mechanics, the buffers are set up correctly
func TestMarketBufCtx(t *testing.T) {
	buf := getMarketBuf()
	buf.cfunc()
}

func TestMarketBufSubUnsub(t *testing.T) {
	buf := getMarketBuf()
	sub1 := buf.Subscribe(1)
	sub2 := buf.Subscribe(1)
	assert.NotNil(t, sub1.Recv())
	assert.NotNil(t, sub2.Recv())
	buf.Unsubscribe(sub1)
	// we expect sub1's Err to return an error indicating the context was cancelled, but sub2 should
	// remain unaffected
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

func TestMarketBufFlush(t *testing.T) {
	buf := getMarketBuf()
	sub := buf.Subscribe(1)
	mkt := types.Market{
		Id:   "test-mkt-1",
		Name: "FOO/BAR",
	}
	buf.Add(mkt)
	buf.Flush()
	subMkt := <-sub.Recv()
	assert.Equal(t, 1, len(subMkt))
	assert.Equal(t, mkt, subMkt[0])
	assert.NoError(t, sub.Err())
	buf.cfunc() // close down everything
	assert.NotNil(t, <-sub.Done())
	assert.Error(t, sub.Err())
}

func getMarketBuf() *mktBufTst {
	ctx, cfunc := context.WithCancel(context.Background())
	return &mktBufTst{
		MarketCh: buffer.NewMarketCh(ctx),
		cfunc:    cfunc,
	}
}
