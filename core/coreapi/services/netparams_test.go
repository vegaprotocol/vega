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

package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/coreapi/services"
	"code.vegaprotocol.io/vega/core/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
