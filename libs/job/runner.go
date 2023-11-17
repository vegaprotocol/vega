// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
