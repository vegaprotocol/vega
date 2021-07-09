package subscribers_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

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

func (m meStub) Type() events.Type {
	return m.t
}

func (m meStub) MarketEvent() string {
	return m.str
}

func (m meStub) SetSequenceID(s uint64) {}
func (m meStub) Sequence() uint64       { return 0 }
