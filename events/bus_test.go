package events_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/events"

	"github.com/stretchr/testify/assert"
)

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
