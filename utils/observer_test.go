package utils_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	logObserver "go.uber.org/zap/zaptest/observer"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
)

func newRecordedLogger() (*logging.Logger, *logObserver.ObservedLogs) {
	zapCore, logs := logObserver.New(zap.DebugLevel)
	zapLogger := zap.New(zapCore)
	logger := &logging.Logger{Logger: zapLogger}
	return logger, logs
}

func TestNotifyDoesNotBlock(t *testing.T) {
	logger, logs := newRecordedLogger()

	// An observer with no input buffer
	o := utils.NewObserver[int]("test", logger, 0, 0)
	ch, _ := o.Observe(context.Background(), 3, func(x int) bool { return true })

	// We have an observer that isn't reading from it's channel - when we notify it should
	// output a debug message saying "channel could not be updated". There's an effective buffer
	// of 1 message in the Observe() select loop, which may or may not have started by the time
	// we Notify(), so notify twice just in case.
	o.Notify([]int{1, 2, 3})
	o.Notify([]int{1, 2, 3})

	flogs := logs.FilterMessageSnippet("channel could not be updated")
	assert.NotZero(t, flogs.Len())

	// And there should be nothing on the channel
	select {
	case <-ch:
		t.Fail()
	default:
	}
}

func TestFilter(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	o := utils.NewObserver[int]("test", logger, 10, 10)
	ch1, _ := o.Observe(ctx, 3, func(x int) bool { return x > 1 })
	ch2, _ := o.Observe(ctx, 3, func(x int) bool { return true })

	o.Notify([]int{1, 2, 3})
	out1 := <-ch1
	out2 := <-ch2

	assert.Equal(t, []int{2, 3}, out1)
	assert.Equal(t, []int{1, 2, 3}, out2)
}

func TestCannotSend(t *testing.T) {
	logger, logs := newRecordedLogger()
	ctx := context.Background()

	// To represent the case where the observer accepts a value on its input channel but
	// cannot output it, create an observer with a small input buffer, but no output buffer
	o := utils.NewObserver[int]("test", logger, 1, 0)
	ch, _ := o.Observe(ctx, 3, func(x int) bool { return true })
	o.Notify([]int{1, 2, 3})

	// The observer goroutine should try 3 times with a short delay between and eventually give up.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, logs.FilterMessageSnippet("retrying").Len())
	assert.Equal(t, 1, logs.FilterMessageSnippet("retry limit").Len())

	// There should be nothing on the channel, and it should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Fail()
		}
	default:
	}
}

func TestCancel(t *testing.T) {
	logger, logs := newRecordedLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	o := utils.NewObserver[int]("test", logger, 0, 0)
	ch, _ := o.Observe(ctx, 3, func(x int) bool { return true })

	// Fire up some goroutines that will pump data through till the end of the test.
	defer pump(&o)()
	defer dump(ch)()

	// Run for a while, then cancel the observer's context
	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)

	// Check that in the logs we got a bunch of successes, then an 'closed connection' message.
	assert.Greater(t, logs.FilterMessageSnippet("sent successfully").Len(), 8)
	assert.Equal(t, logs.FilterMessageSnippet("closed connection").Len(), 1)
}

/******************************************** Helpers ********************************************/

// pump launches a goroutine to notify an observer, 10 time a second
func pump(o *utils.Observer[int]) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.Tick(10 * time.Millisecond)
		for {
			select {
			case <-ticker:
				o.Notify([]int{1, 2, 3})
			case <-ctx.Done():
				return
			}
		}
	}()
	return cancel
}

// dump launches a goroutine that reads from a channel and discards the data
func dump(ch <-chan []int) context.CancelFunc {
	// A goroutine read it out again
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case _, ok := <-ch:
				if !ok {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return cancel
}
