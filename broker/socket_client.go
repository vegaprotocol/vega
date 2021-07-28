package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
)

const (
	dialRetryInterval             = 5 * time.Second
	defaultEventChannelBufferSize = 10000000
)

// SocketClient stream events sent to this broker over a socket to a remote broker.
// This is used to send events from a non-validating core node to a data node.
type socketClient struct {
	ctx context.Context
	log *logging.Logger

	config *SocketConfig
	sock   protocol.Socket

	eventCh  chan []events.Event
	socketCh chan []byte
	errCh    chan error

	reconnectMu  sync.RWMutex
	reconnecting bool
}

func NewSocketClient(ctx context.Context, log *logging.Logger, config *SocketConfig) (SocketClient, error) {
	if !config.Enabled {
		return nil, nil
	}

	sock, err := push.NewSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to create new socket: %w", err)
	}

	s := socketClient{
		ctx: ctx,
		log: log,

		config: config,
		sock:   sock,

		eventCh:  make(chan []events.Event, defaultEventChannelBufferSize),
		socketCh: make(chan []byte, defaultEventChannelBufferSize),
		errCh:    make(chan error),
	}

	if err = s.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	go s.stream()

	return &s, nil
}

func (s socketClient) connect() error {
	var err error
	addr := fmt.Sprintf("tcp://%s:%d", "0.0.0.0", s.config.Port)
	ticker := time.NewTicker(dialRetryInterval)
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case <-time.After(s.config.DialTimeout.Get()):
			return fmt.Errorf("timeout connecting to %v", addr)
		case <-ticker.C:
			if err = s.sock.Dial(addr); err != nil {
				s.log.Error(fmt.Sprintf("failed to connect to %v, retrying", addr), logging.Error(err))
			} else {
				ticker.Stop()
				return nil
			}
		}
	}
}

func (s socketClient) Send(evts []events.Event) {
	s.eventCh <- evts
}

func (s socketClient) Close() error {
	if s.config.Enabled {
		return nil
	}

	<-s.ctx.Done()
	return s.sock.Close()
}

func (s socketClient) reconnect() error {
	s.reconnectMu.Lock()
	s.reconnecting = true
	s.reconnectMu.Unlock()
	defer func() {
		s.reconnectMu.Lock()
		s.reconnecting = false
		s.reconnectMu.Unlock()
	}()

	addr := fmt.Sprintf("tcp://%s:%d", "0.0.0.0", s.config.Port)
	s.log.Warningf(fmt.Sprintf("connection lost to %v, will retry to connect", addr))

	return s.connect()
}

func (s socketClient) stream() {
	for {
		select {
		case <-s.ctx.Done():
			close(s.socketCh)
			close(s.eventCh)
			close(s.errCh)
			return
		case err := <-s.errCh:
			s.log.Error(fmt.Sprintf("terminating event streaming"), logging.Error(err))
			return
		case msg := <-s.socketCh:
			if err := s.sock.Send(msg); err != nil {
				s.socketCh <- msg
				switch err {
				case protocol.ErrClosed:
					s.errCh <- s.reconnect()
				default:
					s.log.Error(fmt.Sprintf("failed to send on socket"), logging.Error(err))
				}
			}
		case evts := <-s.eventCh:
			for _, evt := range evts {
				msg, err := proto.Marshal(evt.StreamMessage())
				if err != nil {
					s.errCh <- fmt.Errorf("fail to marshal event: %v", err)
				}
				s.socketCh <- msg
			}
		default:
		}
	}
}
