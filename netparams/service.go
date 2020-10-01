package netparams

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"
)

type NetParamEvent interface {
	events.Event
	NetParam() types.NetworkParameter
}

type Service struct {
	*subscribers.Base

	params map[string]string
	mu     sync.RWMutex
	ch     chan types.NetworkParameter
}

func NewService(ctx context.Context) *Service {
	return &Service{
		Base:   subscribers.NewBase(ctx, 10, true),
		params: map[string]string{},
	}
}

func (s *Service) GetAll() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]string, len(s.params))
	for k, v := range s.params {
		out[k] = v
	}
	return out
}

func (s *Service) Push(evts ...events.Event) {
	for _, e := range evts {
		if nse, ok := e.(NetParamEvent); ok {
			s.ch <- nse.NetParam()
		}
	}
}

func (s *Service) consume() {
	defer func() { close(s.ch) }()
	for {
		select {
		case <-s.Closed():
			return
		case np, ok := <-s.ch:
			if !ok {
				// cleanup base
				s.Halt()
				// channel is closed
				return
			}
			s.mu.Lock()
			s.params[np.Key] = np.Value
			s.mu.Unlock()
		}
	}
}

func (n *Service) Types() []events.Type {
	return []events.Type{
		events.NetworkParameterEvent,
	}
}
