// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package netparams

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/subscribers"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

type NetParamEvent interface {
	events.Event
	NetworkParameter() types.NetworkParameter
}

type Service struct {
	*subscribers.Base

	params map[string]types.NetworkParameter
	mu     sync.RWMutex
	nch    chan types.NetworkParameter
}

func NewService(ctx context.Context) *Service {
	s := &Service{
		Base:   subscribers.NewBase(ctx, 10, true),
		params: map[string]types.NetworkParameter{},
		nch:    make(chan types.NetworkParameter, 100),
	}

	go s.consume()
	return s
}

// GetAll return the list of all current network parameters
func (s *Service) GetAll() []types.NetworkParameter {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]types.NetworkParameter, 0, len(s.params))
	for _, v := range s.params {
		out = append(out, v)
	}
	return out
}

func (s *Service) Push(evts ...events.Event) {
	for _, e := range evts {
		if nse, ok := e.(NetParamEvent); ok {
			s.nch <- nse.NetworkParameter()
		}
	}
}

func (s *Service) consume() {
	defer func() { close(s.nch) }()
	for {
		select {
		case <-s.Closed():
			return
		case np, ok := <-s.nch:
			if !ok {
				// cleanup base
				s.Halt()
				// channel is closed
				return
			}
			s.mu.Lock()
			s.params[np.Key] = np
			s.mu.Unlock()
		}
	}
}

func (*Service) Types() []events.Type {
	return []events.Type{
		events.NetworkParameterEvent,
	}
}
