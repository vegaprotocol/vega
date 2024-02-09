// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package utils_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	logObserver "go.uber.org/zap/zaptest/observer"
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

	// The channel should be closed eventually, but it might take a few cycles to get there.
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			t.Fail()
		default:
			time.Sleep(10 * time.Millisecond)
		}
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
	time.Sleep(100 * time.Millisecond)
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
