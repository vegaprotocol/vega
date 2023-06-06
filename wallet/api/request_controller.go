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
