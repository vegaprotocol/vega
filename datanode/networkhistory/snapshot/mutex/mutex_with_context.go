package mutex

import "context"

// No decent library found for this, basically lock with context, all libs are overly complex or little used
// Given this, this simple solution from here https://h12.io/article/go-pattern-context-aware-lock does the trick

type CtxMutex interface {
	Lock(ctx context.Context) bool
	Unlock()
}

type ctxMutex struct {
	ch chan struct{}
}

func New() CtxMutex {
	return &ctxMutex{
		ch: make(chan struct{}, 1),
	}
}

func (mu *ctxMutex) Lock(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case mu.ch <- struct{}{}:
		return true
	}
}

func (mu *ctxMutex) Unlock() {
	select {
	case <-mu.ch:
	default:
	}
}
