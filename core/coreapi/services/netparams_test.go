package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/core/events"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/core/coreapi/services"
)

func TestNetParams(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	np := services.NewNetParams(ctx)
	wg := sync.WaitGroup{}
	wg.Add(1)
	allSent := false

	maxEvents := 1000000

	evts := make([]events.Event, maxEvents)

	for i := 0; i < maxEvents; i++ {
		evts[i] = events.NewNetworkParameterEvent(ctx, "foo", "bar")
	}

	require.NotPanics(t, func() {
		go func() {
			np.Push(
				evts...,
			)
			allSent = true
			wg.Done()
		}()
	})

	// slight pause to give the goroutine a chance to start pushing before we cancel the context
	time.Sleep(time.Millisecond)
	cancel()

	wg.Wait()

	assert.True(t, allSent)
}
