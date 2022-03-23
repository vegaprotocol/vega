package broker

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/libs/proto"
	mangos "go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/inproc"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"

	"code.vegaprotocol.io/vega/events"
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

func newSocketClient(ctx context.Context, log *logging.Logger, config *SocketConfig) (*socketClient, error) {
	sock, err := push.NewSocket()
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
		log: log,

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

	var sendTimeouts int

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
					sendTimeouts++
					s.log.Error("Failed to queue message on socket", logging.Error(err))

					if sendTimeouts > s.config.MaxSendTimeouts {
						return fmt.Errorf(
							"maximum number of '%d' send timeouts exceeded: %w",
							s.config.MaxSendTimeouts,
							err,
						)
					}

					// Try to put the timed out message back on internal events queue
					if err := s.Send(evt); err != nil {
						return err
					}
				default:
					s.log.Error("Failed to send to socket", logging.Error(err))
				}

				continue
			}

			sendTimeouts = 0
		}
	}
}
