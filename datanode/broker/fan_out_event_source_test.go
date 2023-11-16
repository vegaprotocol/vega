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

package broker_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/service"
	vgcontext "code.vegaprotocol.io/vega/libs/context"

	"github.com/stretchr/testify/assert"
)

func TestEventFanOut(t *testing.T) {
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error),
	}
	ctx := vgcontext.WithBlockHeight(context.Background(), 1)
	fos := broker.NewFanOutEventSource(tes, 20, 2)

	evtCh1, _ := fos.Receive(ctx)
	evtCh2, _ := fos.Receive(ctx)

	e1 := events.NewAssetEvent(ctx, types.Asset{ID: "a1"})
	e2 := events.NewAssetEvent(ctx, types.Asset{ID: "a2"})
	e1.SetSequenceID(1)
	e2.SetSequenceID(2)

	tes.eventsCh <- e1
	tes.eventsCh <- e2

	assert.Equal(t, e1, <-evtCh1)
	assert.Equal(t, e1, <-evtCh2)

	assert.Equal(t, e2, <-evtCh1)
	assert.Equal(t, e2, <-evtCh2)
}

func TestCompositeFanOut(t *testing.T) {
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error),
	}
	c, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	ctx := vgcontext.WithBlockHeight(c, 1)
	fos := broker.NewFanOutEventSource(tes, 20, 1)

	evtCh1, _ := fos.Receive(ctx)

	e1 := events.NewAssetEvent(ctx, types.Asset{ID: "a1"})
	e2 := events.NewExpiredOrdersEvent(ctx, "foo", []string{
		"party1",
		"party2",
		"party3",
		"party4",
	})
	e3 := events.NewAssetEvent(ctx, types.Asset{ID: "a2"})
	e4 := events.NewAssetEvent(ctx, types.Asset{ID: "a3"})
	// set seq ID as expected
	sID := uint64(1)
	e1.SetSequenceID(sID)
	sID += e1.CompositeCount()
	e2.SetSequenceID(sID)
	sID += e2.CompositeCount()
	e3.SetSequenceID(sID)
	sID += e3.CompositeCount()
	e4.SetSequenceID(sID)

	tes.eventsCh <- e1
	tes.eventsCh <- e2
	tes.eventsCh <- e3
	tes.eventsCh <- e4

	assert.Equal(t, e1, <-evtCh1)
	assert.Equal(t, e2, <-evtCh1)
	assert.Equal(t, e3, <-evtCh1)
	assert.Equal(t, e4, <-evtCh1)
}

func TestCloseChannelsAndExitWithError(t *testing.T) {
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error, 1),
	}

	ctx := vgcontext.WithBlockHeight(context.Background(), 1)
	fos := broker.NewFanOutEventSource(tes, 20, 2)

	evtCh1, errCh1 := fos.Receive(ctx)
	evtCh2, errCh2 := fos.Receive(ctx)

	e := events.NewAssetEvent(ctx, types.Asset{ID: "a1"})
	e.SetSequenceID(1)
	tes.eventsCh <- e
	assert.Equal(t, e, <-evtCh1)
	assert.Equal(t, e, <-evtCh2)

	tes.errorsCh <- fmt.Errorf("e1")
	close(tes.eventsCh)

	assert.Equal(t, fmt.Errorf("e1"), <-errCh1)
	assert.Equal(t, fmt.Errorf("e1"), <-errCh2)

	_, ok := <-evtCh1
	assert.False(t, ok, "channel should be closed")
	_, ok = <-evtCh2
	assert.False(t, ok, "channel should be closed")
}

func TestPanicOnInvalidSubscriberNumber(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()

	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error),
	}

	fos := broker.NewFanOutEventSource(tes, 20, 2)

	fos.Receive(context.Background())
	fos.Receive(context.Background())
	fos.Receive(context.Background())
}

func TestListenOnlyCalledOnceOnSource(t *testing.T) {
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error),
	}

	fos := broker.NewFanOutEventSource(tes, 20, 2)
	fos.Listen()
	fos.Listen()
	fos.Listen()

	assert.Equal(t, 1, tes.listenCount)
}

type testEventSource struct {
	eventsCh           chan events.Event
	errorsCh           chan error
	listenCount        int
	protocolUpgradeSvc *service.ProtocolUpgrade
}

func (te *testEventSource) Listen() error {
	te.listenCount++
	return nil
}

func (te *testEventSource) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	return te.eventsCh, te.errorsCh
}

func (te *testEventSource) Send(e events.Event) error {
	return nil
}
