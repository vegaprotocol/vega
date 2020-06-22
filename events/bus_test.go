package events_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/events"

	"github.com/stretchr/testify/assert"
)

func getCtx() context.Context {
	return contextutil.WithTraceID(context.Background(), "test-trace-id")
}

func TestTimeEvent(t *testing.T) {
	now := time.Now()
	ctx := context.Background()
	e := events.NewTime(ctx, now)
	assert.Equal(t, e.Time(), now)
	assert.Equal(t, events.TimeUpdate, e.Type())
	assert.NotEmpty(t, e.TraceID())
	_, trace := contextutil.TraceIDFromContext(e.Context())
	assert.NotNil(t, trace)
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
