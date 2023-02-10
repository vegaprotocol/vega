// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package broker

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"go.nanomsg.org/mangos/v3"
	mangosErr "go.nanomsg.org/mangos/v3/errors"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pair"
	_ "go.nanomsg.org/mangos/v3/transport/inproc" // Does some nanomsg magic presumably
	_ "go.nanomsg.org/mangos/v3/transport/tcp"    // Does some nanomsg magic presumably
	"golang.org/x/sync/errgroup"

	"code.vegaprotocol.io/vega/libs/proto"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/logging"
)

// SocketClient stream events sent to this broker over a socket to a remote broker.
// This is used to send events from a non-validating core node to a data node.
type socketClient struct {
	log *logging.Logger

	config *SocketConfig
	sock   protocol.Socket

	eventsCh chan events.Event

	closed bool

	mut sync.RWMutex
}

func pipeEventToString(pe mangos.PipeEvent) string {
	switch pe {
	case mangos.PipeEventAttached:
		return "Attached"
	case mangos.PipeEventDetached:
		return "Detached"
	default:
		return "Attaching"
	}
}

const namedSocketClientLogger = "socket-client"

func newSocketClient(ctx context.Context, log *logging.Logger, config *SocketConfig) (*socketClient, error) {
	sock, err := pair.NewSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to create new push socket: %w", err)
	}

	socOpts := map[string]interface{}{
		mangos.OptionWriteQLen:    config.SocketChannelBufferSize,
		mangos.OptionSendDeadline: config.SocketQueueTimeout.Duration,
	}

	for name, value := range socOpts {
		if err := sock.SetOption(name, value); err != nil {
			return nil, fmt.Errorf("failed to set option: %w", err)
		}
	}

	sock.SetPipeEventHook(func(pe mangos.PipeEvent, p mangos.Pipe) {
		log.Info(
			"New broker connection event",
			logging.String("eventType", pipeEventToString(pe)),
			logging.Uint32("id", p.ID()),
			logging.String("address", p.Address()),
		)
	})

	s := &socketClient{
		log: log.Named(namedSocketClientLogger),

		config: config,
		sock:   sock,

		eventsCh: make(chan events.Event, config.EventChannelBufferSize),
	}

	if err := s.connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	go func() {
		if err := s.stream(ctx); err != nil {
			s.log.Fatal("socket streaming has failed", logging.Error(err))
		}
	}()

	return s, nil
}

func (s *socketClient) SendBatch(evts []events.Event) error {
	for _, evt := range evts {
		if err := s.Send(evt); err != nil {
			return err
		}
	}

	return nil
}

// Send sends events on the events queue.
// Panics if socket is closed or is not streaming.
// Returns an error if events queue is full.
func (s *socketClient) Send(evt events.Event) error {
	s.mut.RLock()
	defer s.mut.RUnlock()

	if s.closed {
		s.log.Panic("Failed to send event - socket is closed closed socket")
	}

	select {
	case s.eventsCh <- evt:
		break
	case <-time.After(2 * time.Second):
	default:
		return fmt.Errorf("event queue is full")
	}

	return nil
}

func (s *socketClient) close() {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.closed = true
	close(s.eventsCh)

	if err := s.sock.Close(); err != nil {
		s.log.Error("failed to close socket", logging.Error(err))
	}
}

func (s *socketClient) getDialAddr() string {
	return fmt.Sprintf(
		"%s://%s",
		strings.ToLower(s.config.Transport),
		net.JoinHostPort(s.config.Address, fmt.Sprintf("%d", s.config.Port)),
	)
}

func (s *socketClient) connect(ctx context.Context) error {
	ticker := time.NewTicker(s.config.DialRetryInterval.Get())
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(ctx, s.config.DialTimeout.Get())
	defer cancel()

	addr := s.getDialAddr()

	for {
		err := s.sock.Dial(addr)
		if err == nil {
			return nil
		}

		s.log.Error("failed to connect, retrying", logging.Error(err), logging.String("peer", addr))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *socketClient) stream(ctx context.Context) error {
	s.mut.RLock()
	if s.closed {
		s.mut.RUnlock()
		return fmt.Errorf("socket is closed")
	}
	s.mut.RUnlock()

	defer s.close()

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt := <-s.eventsCh:
			msg, err := proto.Marshal(evt.StreamMessage())
			if err != nil {
				s.log.Error("Failed to marshal event", logging.Error(err))

				continue
			}

			err = s.sock.Send(msg)
			if err != nil {
				switch err {
				case protocol.ErrClosed:
					return fmt.Errorf("socket is closed: %w", err)
				case protocol.ErrSendTimeout:
					return fmt.Errorf("failed to queue message on socket: %w", err)
				default:
					s.log.Error("Failed to send to socket", logging.Error(err))
				}

				continue
			}
		}
	}
}

func (s *socketClient) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	// channel onto which we push the raw messages from the queue
	inboundCh := make(chan []byte, 10)
	stopCh := make(chan struct{}, 1)

	outboundCh := make(chan events.Event, 10)
	errCh := make(chan error, 1)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()

		return nil
	})

	eg.Go(func() error {
		defer close(outboundCh)

		for msg := range inboundCh {
			var be eventspb.BusEvent
			if err := proto.Unmarshal(msg, &be); err != nil {
				// surely we should stop if this happens?
				s.log.Error("Failed to unmarshal received event", logging.Error(err))
				continue
			}
			if be.Version != eventspb.Version {
				return fmt.Errorf("mismatched BusEvent version received: %d, want %d", be.Version, eventspb.Version)
			}

			evt := toEvent(ctx, &be)
			if evt == nil {
				s.log.Error("Can not convert proto event to internal event", logging.String("event_type", be.GetType().String()))
				continue
			}

			// Listen for context cancels, even if we're blocked sending events
			select {
			case outboundCh <- evt:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return nil
	})

	eg.Go(func() error {
		defer close(inboundCh)

		s.sock.SetOption(mangos.OptionRecvDeadline, 1*time.Second)
		for {
			msg, err := s.sock.Recv()
			if err != nil {
				switch err {
				case mangosErr.ErrRecvTimeout:
					select {
					case <-stopCh:
						return nil
					default:
					}
				case mangosErr.ErrBadVersion:
					return fmt.Errorf("failed with bad protocol version: %w", err)
				case mangosErr.ErrClosed:
					return nil
				default:
					s.log.Error("Failed to Receive message", logging.Error(err))
					continue
				}
			}

			if len(msg) == 0 {
				continue
			}

			inboundCh <- msg
		}
	})

	go func() {
		defer func() {
			close(errCh)
		}()

		if err := eg.Wait(); err != nil {
			errCh <- err
		}
	}()

	return outboundCh, errCh
}
