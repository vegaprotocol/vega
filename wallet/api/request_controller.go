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

package api

import (
	"fmt"
	"sync"
	"time"
)

type RequestController struct {
	publicKeysInUse sync.Map

	maximumAttempt              uint
	intervalDelayBetweenRetries time.Duration
}

func (c *RequestController) IsPublicKeyAlreadyInUse(publicKey string) (func(), error) {
	doneCh, err := c.impatientWait(publicKey)
	if err != nil {
		return nil, err
	}

	go func() {
		<-doneCh
		c.publicKeysInUse.Delete(publicKey)
	}()

	return func() {
		close(doneCh)
	}, nil
}

func (c *RequestController) impatientWait(publicKey string) (chan interface{}, error) {
	tick := time.NewTicker(c.intervalDelayBetweenRetries)
	defer tick.Stop()

	doneCh := make(chan interface{})

	attemptsLeft := c.maximumAttempt
	for attemptsLeft > 0 {
		if _, alreadyInUse := c.publicKeysInUse.LoadOrStore(publicKey, doneCh); !alreadyInUse {
			return doneCh, nil
		}
		<-tick.C
		attemptsLeft--
	}

	close(doneCh)
	return nil, fmt.Errorf("this public key %q is already in use, retry later", publicKey)
}

func DefaultRequestController() *RequestController {
	return NewRequestController(
		WithMaximumAttempt(10),
		WithIntervalDelayBetweenRetries(2*time.Second),
	)
}

func NewRequestController(opts ...RequestControllerOptionFn) *RequestController {
	rq := &RequestController{
		publicKeysInUse: sync.Map{},
	}

	for _, opt := range opts {
		opt(rq)
	}

	return rq
}

type RequestControllerOptionFn func(rq *RequestController)

func WithMaximumAttempt(max uint) RequestControllerOptionFn {
	return func(rq *RequestController) {
		rq.maximumAttempt = max
	}
}

func WithIntervalDelayBetweenRetries(duration time.Duration) RequestControllerOptionFn {
	return func(rq *RequestController) {
		rq.intervalDelayBetweenRetries = duration
	}
}
