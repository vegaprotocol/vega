// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package subscribers_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"github.com/golang/mock/gomock"
)

type tstME struct {
	*subscribers.MarketEvent
	ctx   context.Context
	cfunc context.CancelFunc
	ctrl  *gomock.Controller
}

type meStub struct {
	str string
	t   events.Type
}

func getTestME(t *testing.T, ack bool) *tstME {
	ctrl := gomock.NewController(t)
	ctx, cfunc := context.WithCancel(context.Background())
	return &tstME{
		MarketEvent: subscribers.NewMarketEvent(ctx, subscribers.NewDefaultConfig(), logging.NewTestLogger(), ack),
		ctx:         ctx,
		cfunc:       cfunc,
		ctrl:        ctrl,
	}
}

func (t *tstME) Finish() {
	t.cfunc()
	t.ctrl.Finish()
}

func TestPush(t *testing.T) {
	t.Run("Test push market event - success", testPushSuccess)
	t.Run("Test push with a non-market event is ignored", testPushIgnore)
}

func testPushSuccess(t *testing.T) {
	me := getTestME(t, true)
	defer me.Finish()
	for _, et := range events.MarketEvents() {
		e := meStub{
			str: fmt.Sprintf("test event %s", et.String()),
			t:   et,
		}
		me.Push(e)
	}
}

func testPushIgnore(t *testing.T) {
	me := getTestME(t, true)
	defer me.Finish()
	// this is not a market event
	e := trStub{
		r: []*types.TransferResponse{},
	}
	me.Push(e)
}

func (m meStub) Context() context.Context {
	return context.TODO()
}

func (m meStub) TraceID() string {
	return "trace-id-test"
}

func (m meStub) TxHash() string {
	return "txhash-test"
}

func (m meStub) Type() events.Type {
	return m.t
}

func (m meStub) MarketEvent() string {
	return m.str
}

func (m meStub) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{}
}

func (m meStub) SetSequenceID(s uint64) {}
func (m meStub) Sequence() uint64       { return 0 }
func (m meStub) BlockNr() int64         { return 0 }
func (m meStub) ChainID() string        { return "testchain" }
