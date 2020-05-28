package events_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"

	"github.com/stretchr/testify/assert"
)

func getCtx() context.Context {
	ctx := context.WithValue(context.Background(), events.TraceIDKey, "test-trace-id")
	return ctx
}

func TestTimeEvent(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	e := events.NewTime(ctx, now)
	assert.Equal(t, e.Time(), now)
	assert.Equal(t, events.TimeUpdate, e.Type())
	assert.NotEmpty(t, e.TraceID())
	tID := e.Context().Value(events.TraceIDKey)
	assert.NotNil(t, tID)
	trace, ok := tID.(string)
	assert.True(t, ok)
	assert.Equal(t, trace, e.TraceID())
	assert.Zero(t, e.Sequence())
}

func TestGenericEvent(t *testing.T) {
	now := time.Now()
	ctx := getCtx()
	ge, err := events.New(ctx, now)
	assert.NoError(t, err)
	e, ok := ge.(*events.Time)
	assert.True(t, ok)
	assert.Equal(t, now, e.Time())
	assert.Equal(t, events.TimeUpdate, e.Type())
	// try same with time pointer
	ge, err = events.New(ctx, &now)
	assert.NoError(t, err)
	e2, ok := ge.(*events.Time)
	assert.True(t, ok)
	assert.Equal(t, e.Time(), e2.Time())
}

func TestInvalidEvent(t *testing.T) {
	_, err := events.New(context.Background(), events.TimeUpdate)
	assert.Error(t, err)
	assert.Equal(t, events.ErrUnsuportedEvent, err)
}

func TestInvalidTraceIDType(t *testing.T) {
	tIDInt := 123
	ctx := context.WithValue(context.Background(), events.TraceIDKey, tIDInt) // int instead of string
	e := events.NewTime(ctx, time.Now())
	assert.NotEqual(t, e.TraceID(), tIDInt)
	assert.NotEqual(t, ctx.Value(events.TraceIDKey), e.Context().Value(events.TraceIDKey))
}
