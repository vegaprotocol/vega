package sqlsubscribers

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/protos/vega"
	"github.com/stretchr/testify/assert"
)

func TestEventDeduplicator_Flush(t *testing.T) {
	edd := NewEventDeduplicator[string, *vega.LiquidityProvision](func(ctx context.Context,
		lp *vega.LiquidityProvision, vegaTime time.Time) (string, error) {
		return lp.Id, nil
	})

	lp1 := &vega.LiquidityProvision{
		Id: "1",
	}

	edd.AddEvent(context.Background(), lp1, time.Now())
	events := edd.Flush()
	assert.Equal(t, lp1, events["1"])

	lp2 := &vega.LiquidityProvision{
		Id:     "1",
		Status: vega.LiquidityProvision_STATUS_PENDING,
	}

	edd.AddEvent(context.Background(), lp2, time.Now())
	events = edd.Flush()
	assert.Equal(t, lp2, events["1"])

	edd.AddEvent(context.Background(), lp2, time.Now())
	events = edd.Flush()
	assert.Equal(t, 0, len(events))

	edd.AddEvent(context.Background(), lp2, time.Now())
	edd.AddEvent(context.Background(), lp1, time.Now())
	edd.AddEvent(context.Background(), lp2, time.Now())
	events = edd.Flush()
	assert.Equal(t, 0, len(events))

	edd.AddEvent(context.Background(), lp1, time.Now())
	edd.AddEvent(context.Background(), lp2, time.Now())
	edd.AddEvent(context.Background(), lp1, time.Now())
	events = edd.Flush()
	assert.Equal(t, lp1, events["1"])
}
