package buffer

import "context"

type subscriber struct {
	ctx context.Context
}

func (s subscriber) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s subscriber) Err() error {
	return s.ctx.Err()
}
