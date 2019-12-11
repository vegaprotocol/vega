package buffer

import "context"

type subscriber struct {
	ctx   context.Context
	cfunc context.CancelFunc
	key   int
}

func (s subscriber) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s subscriber) Err() error {
	return s.ctx.Err()
}
