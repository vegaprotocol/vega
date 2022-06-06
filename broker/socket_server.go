package broker

import (
	"context"
	"fmt"
	"net"
	"strings"

	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"golang.org/x/sync/errgroup"

	"github.com/golang/protobuf/proto"
	mangos "go.nanomsg.org/mangos/v3"
	mangosErr "go.nanomsg.org/mangos/v3/errors"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	_ "go.nanomsg.org/mangos/v3/transport/inproc"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
)

// socketServer receives events from a remote broker.
// This is used by the data node to receive events from a non-validating core node.
type socketServer struct {
	log    *logging.Logger
	config *Config

	sock protocol.Socket
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

func newSocketServer(log *logging.Logger, config *Config) (*socketServer, error) {
	sock, err := pull.NewSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to create new socket: %w", err)
	}

	return &socketServer{
		log:    log,
		config: config,
		sock:   sock,
	}, nil
}

func (s socketServer) Listen() error {
	addr := fmt.Sprintf(
		"%s://%s",
		strings.ToLower(s.config.SocketConfig.TransportType),
		net.JoinHostPort(s.config.SocketConfig.IP, fmt.Sprintf("%d", s.config.SocketConfig.Port)),
	)

	if err := s.sock.Listen(addr); err != nil {
		return fmt.Errorf("failed to listen on %v: %w", addr, err)
	}

	s.log.Info("Starting broker socket server", logging.String("addr", s.config.SocketConfig.IP),
		logging.Int("port", s.config.SocketConfig.Port))

	s.sock.SetPipeEventHook(func(pe mangos.PipeEvent, p mangos.Pipe) {
		s.log.Info(
			"New broker connection event",
			logging.String("eventType", pipeEventToString(pe)),
			logging.Uint32("id", p.ID()),
			logging.String("address", p.Address()),
		)
	})

	return nil
}

func (s socketServer) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	outboundCh := make(chan events.Event, s.config.SocketServerOutboundBufferSize)
	// channel onto which we push the raw messages from the queue
	inboundCh := make(chan []byte, s.config.SocketServerInboundBufferSize)
	errCh := make(chan error, 1)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()
		if err := s.close(); err != nil {
			return fmt.Errorf("failed to close socket: %w", err)
		}
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
		var recvTimeouts int
		defer close(inboundCh)
		for {
			msg, err := s.sock.Recv()
			if err != nil {
				switch err {
				case mangosErr.ErrRecvTimeout:
					s.log.Warn("Receive socket timeout", logging.Error(err))
					recvTimeouts++
					if recvTimeouts > s.config.SocketConfig.MaxReceiveTimeouts {
						return fmt.Errorf("more then a 3 socket timeouts occurred: %w", err)
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

			inboundCh <- msg
			recvTimeouts = 0
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

func (s socketServer) close() error {
	s.log.Info("Closing socket server")
	return s.sock.Close()
}
