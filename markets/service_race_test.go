// +build !race

package markets_test

import (
	"sync"
	"testing"

	types "code.vegaprotocol.io/vega/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMarketObserveDepthRetryLimit(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	marketArg := "TSTmarket"
	retries := 2
	ref := uint64(1)
	// return value of GetMarketDepth call
	depth := types.MarketDepth{
		MarketId: marketArg,
	}
	// ensure unsubscribe was handled properly
	wg := sync.WaitGroup{}
	wg.Add(1)

	// perform this write in a routine, somehow this doesn't work when we use an anonymous func in the Do argument
	writeToChannel := func(ch chan<- []types.Order) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Recovered from panic (possibly write to closed chan): %+v", r)
			}
		}()
		// keep writing to channel until context is cancelled
		for {
			select {
			case <-svc.ctx.Done():
				return
			case ch <- []types.Order{}:
				continue
			}
		}
	}
	svc.order.EXPECT().Subscribe(gomock.Any()).Times(1).Return(ref).Do(func(ch chan<- []types.Order) {
		go writeToChannel(ch)
	})

	// at least == to number of retries
	svc.marketDepth.EXPECT().GetMarketDepth(gomock.Any(), marketArg, uint64(0)).MinTimes(1).Return(&depth, nil)
	// waitgroup here ensures that unsubscribe was indeed called
	svc.order.EXPECT().Unsubscribe(ref).Times(1).DoAndReturn(func(id uint64) error {
		svc.cfunc() // cancel writeToChannel
		wg.Done()
		assert.Equal(t, id, ref)
		return nil
	})

	// we're ignoring the channel, this is testing the retry limit
	_, id := svc.ObserveDepth(svc.ctx, retries, marketArg)
	assert.Equal(t, ref, id) // should be returned straight from the orderStore mock
	// we're not reading from the return channel, we're waiting for retry limit to be reached instead
	wg.Wait() // wait for unsubscribe call
}
