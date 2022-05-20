package sqlsubscribers

import (
	"context"
	"time"
)

type subscriber struct {
	vegaTime time.Time
}

func (s *subscriber) SetVegaTime(vegaTime time.Time) {
	s.vegaTime = vegaTime
}

func (s *subscriber) Flush(ctx context.Context) error {
	return nil
}
