package broker

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/data-node/proto/events/v1"

	"github.com/golang/protobuf/proto"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
)

const (
	defaultMaxRetries    = 10
	defaultRetryInternal = 50 * time.Millisecond
)

// SocketServer receives events from a remote broker.
// This is used by the data node to receive events from a non-validating core node.
type SocketServer struct {
	ctx context.Context

	log  *logging.Logger
	sock protocol.Socket

	quit chan struct{}
}

func NewSocketServer(ctx context.Context, log *logging.Logger, config *SocketConfig) (*SocketServer, error) {
	sock, err := pull.NewSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to create new socket: %w", err)
	}

	addr := fmt.Sprintf("tcp://%s:%d", config.IP, config.Port)

	if err = sock.Listen(addr); err != nil {
		return nil, fmt.Errorf("failed to listen on %v: %w", addr, err)
	}

	return &SocketServer{
		ctx:  ctx,
		log:  log,
		sock: sock,
		quit: make(chan struct{}),
	}, nil
}

func (s SocketServer) Receive(ctx context.Context, ch chan events.Event) {
	var err error
	var msg []byte

	var be eventspb.BusEvent
	retryCount := defaultMaxRetries
	for {
		msg, err = s.sock.Recv()
		if err != nil {
			switch err {
			case protocol.ErrRecvTimeout:
				if retryCount > 0 {
					retryCount--
					time.Sleep(defaultRetryInternal)
					s.log.Warningf("timeout receiving from socket, retrying", logging.Int("retry-count", retryCount))
					continue
				}
				s.log.Error("socket timeout, stopping event stream", logging.Error(err))
				close(s.quit)
				return
			case protocol.ErrClosed:
				s.log.Error("socket closed, stopping event stream", logging.Error(err))
				close(s.quit)
				return
			default:
				s.log.Error("failed to receive message", logging.Error(err))
			}
			continue
		}

		err = proto.Unmarshal(msg, &be)
		if err != nil {
			s.log.Fatal("failed to unmarshal event received", logging.Error(err))
		}

		evt := toEvent(ctx, &be)
		ch <- evt

		retryCount = defaultMaxRetries
	}
}

func (s *SocketServer) Quit() <-chan struct{} {
	return s.quit
}

func (s SocketServer) Close() error {
	select {
	case <-s.ctx.Done():
	case <-s.quit:
	}
	return s.sock.Close()
}
