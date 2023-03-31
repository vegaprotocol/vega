package job

import (
	"context"
	"sync"
	"sync/atomic"
)

type Runner struct {
	wg           sync.WaitGroup
	jobsCtx      context.Context
	jobsCancelFn context.CancelFunc

	// blown tells if the runner has been used end-to-end (Go + StopAllJobs), or
	// not. If blown, the runner can't be used anymore.
	blown atomic.Bool
}

func (r *Runner) Ctx() context.Context {
	return r.jobsCtx
}

func (r *Runner) Go(fn func(ctx context.Context)) {
	if r.blown.Load() {
		panic("the Runner cannot be recycled")
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		fn(r.jobsCtx)
	}()
}

func (r *Runner) StopAllJobs() {
	r.blown.Store(true)
	r.jobsCancelFn()
	r.wg.Wait()
}

func NewRunner(ctx context.Context) *Runner {
	jobsCtx, jobsCancelFn := context.WithCancel(ctx)
	return &Runner{
		blown:        atomic.Bool{},
		jobsCtx:      jobsCtx,
		jobsCancelFn: jobsCancelFn,
		wg:           sync.WaitGroup{},
	}
}
