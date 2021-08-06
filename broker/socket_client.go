package broker

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	mangos "go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/push"
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

func NewSocketClient(ctx context.Context, log *logging.Logger, config *SocketConfig) (SocketClient, error) {
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

	s := socketClient{
		log: log,

		config: config,
		sock:   sock,

		eventsCh: make(chan events.Event, config.EventChannelBufferSize),
	}

	if err := s.connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	go s.stream(ctx)

	return &s, nil
}

func (s *socketClient) SendBatch(evts []events.Event) {
	for _, evt := range evts {
		s.Send(evt)
	}
}

func (s *socketClient) Send(evts events.Event) {
	select {
	case s.eventsCh <- evts:
		break
	default:
		s.log.Error("Fucked up man - channel is closed")
	}
}

func (s *socketClient) close() error {
	return s.sock.Close()
}

func (s *socketClient) getDialAddr() string {
	return fmt.Sprintf(
		"%s://%s",
		strings.ToLower(s.config.Transport),
		net.JoinHostPort(s.config.IP, fmt.Sprintf("%d", s.config.Port)),
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

		s.log.Error(fmt.Sprintf("failed to connect to %v, retrying", addr), logging.Error(err))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *socketClient) stream(ctx context.Context) {
	var sendTimeouts int

	for {
		select {
		case <-ctx.Done():
			if err := s.close(); err != nil {
				s.log.Error("failed to close socket", logging.Error(err))
			}
			return
		case evt := <-s.eventsCh:
			msg, err := proto.Marshal(evt.StreamMessage())
			if err != nil {
				s.log.Error("fail to marshal event", logging.Error(err))
				continue
			}

			err = s.sock.Send(msg)
			if err != nil {
				switch err {
				case protocol.ErrClosed:
					return
				case protocol.ErrSendTimeout:
					sendTimeouts++
					s.log.Error("failed to queue message on socket", logging.Error(err))

					if sendTimeouts > s.config.MaxSendTimeouts {
						msg := fmt.Sprintf("maximum number '%d' of send timeouts exceeded", s.config.MaxSendTimeouts)
						s.log.Error(msg, logging.Error(err))
						return
					}
				}

				s.log.Error("failed to send to socket", logging.Error(err))
			}
		}
	}
}
