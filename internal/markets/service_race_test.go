// +build !race

package markets

import (
	"context"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage/newmocks"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMarketObserveDepthRetryLimit(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	mockCtrl := gomock.NewController(t)
	marketStore := newmocks.NewMockMarketStore(mockCtrl)
	orderStore := newmocks.NewMockOrderStore(mockCtrl)
	marketArg := "TSTmarket"
	retries := 2
	ref := uint64(1)
	// return value of GetMarketDepth call
	depth := types.MarketDepth{
		Name: marketArg,
	}
	// ensure unsubscribe was handled properly
	wg := sync.WaitGroup{}
	wg.Add(1)

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	conf := NewDefaultConfig(logger)
	marketService, err := NewMarketService(conf, marketStore, orderStore)
	assert.Nil(t, err)
	// set up calls

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
			case <-ctx.Done():
				return
			case ch <- []types.Order{}:
				continue
			}
		}
	}
	orderStore.EXPECT().Subscribe(gomock.Any()).Times(1).Return(ref).Do(func(ch chan<- []types.Order) {
		go writeToChannel(ch)
	})

	// at least == to number of retries
	orderStore.EXPECT().GetMarketDepth(gomock.Any(), marketArg).MinTimes(retries).Return(&depth, nil)
	// waitgroup here ensures that unsubscribe was indeed called
	orderStore.EXPECT().Unsubscribe(ref).Times(1).DoAndReturn(func(id uint64) error {
		cfunc() // cancel writeToChannel
		wg.Done()
		assert.Equal(t, id, ref)
		return nil
	})

	// we're ignoring the channel, this is testing the retry limit
	_, id := marketService.ObserveDepth(ctx, retries, marketArg)
	assert.Equal(t, ref, id) // should be returned straight from the orderStore mock
	// we're not reading from the return channel, we're waiting for retry limit to be reached instead
	wg.Wait() // wait for unsubscribe call
	// end mocks
	mockCtrl.Finish()
}
