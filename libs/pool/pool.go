// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package pool

import "sync"

// Pool is a convenience wrapper around the standard sync.Pool.
// Its main purpose is to avoid cluttering the code with type assertions.
type Pool[T any] struct {
	p *sync.Pool
}

func New[T any](newF func() T) *Pool[T] {
	return &Pool[T]{
		p: &sync.Pool{
			New: func() any {
				return newF()
			},
		},
	}
}

// Get wraps around the standard sync.Get call, but takes care of the type assertion for you.
func (p *Pool[T]) Get() T {
	a := p.p.Get().(T)
	return a
}

// Put adds a given item back to the underlying pool for later use.
// Unlike the underlying sync.Pool.Put function, generics ensure the type is correct, too.
func (p *Pool[T]) Put(i T) {
	p.p.Put(i)
}
