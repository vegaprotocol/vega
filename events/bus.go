package events

import (
	"context"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	ErrUnsuportedEvent = errors.New("unknown payload for event")
)

type Type int

// Base common denominator all event-bus events share
type Base struct {
	ctx     context.Context
	traceID string
	seq     int
	et      Type
}

type Event interface {
	Type() Type
	Context() context.Context
	TraceID() string
}

const (
	// All event type -> used by subscrubers to just receive all events, has no actual corresponding event payload
	All Type = iota
	// other event types that DO have corresponding event types
	TimeUpdate
	TransferResponses
)

// New is a generic constructor - based on the type of v, the specific event will be returned
func New(ctx context.Context, v interface{}) (interface{}, error) {
	switch tv := v.(type) {
	case *time.Time:
		e := NewTime(ctx, *tv)
		return e, nil
	case time.Time:
		e := NewTime(ctx, tv)
		return e, nil
	case []*types.TransferResponse:
		e := NewTransferResponse(ctx, tv)
		return e, nil
	}
	return nil, ErrUnsuportedEvent
}

// A base event holds no data, so the constructor will not be called directly
func newBase(ctx context.Context, t Type) *Base {
	b := Base{
		ctx: ctx,
		et:  t,
	}
	tID := ctx.Value("traceID")
	if tID == nil {
		b.traceID = uuid.NewV4().String()
		ctx = context.WithValue(ctx, "traceID", b.traceID)
		b.ctx = ctx
	} else if s, ok := tID.(string); !ok {
		b.traceID = uuid.NewV4().String()
		ctx = context.WithValue(ctx, "traceID", b.traceID)
		b.ctx = ctx
	} else {
		b.traceID = s
	}
	return &b
}

// TraceID returns the... traceID obviously
func (b Base) TraceID() string {
	return b.traceID
}

// Sequence returns event sequence number
func (b Base) Sequence() int {
	return b.seq
}

// Context returns context
func (b Base) Context() context.Context {
	return b.ctx
}

// Type returns the event type
func (b Base) Type() Type {
	return b.et
}
