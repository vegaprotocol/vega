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
	}, nil

}

func (s SocketServer) Receive(ctx context.Context, ch chan events.Event) {
	var err error
	var msg []byte

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
				}
			case protocol.ErrClosed:
				s.log.Fatal("event socket closed", logging.Error(err))
			default:
				s.log.Error("failed to receive message", logging.Error(err))
			}
			continue
		}

		var be eventspb.BusEvent
		err = proto.Unmarshal(msg, &be)
		if err != nil {
			s.log.Fatal("failed to unmarshal event received", logging.Error(err))
		}

		evt := toEvent(ctx, &be)
		ch <- evt

		retryCount = defaultMaxRetries
	}
}

func (s SocketServer) Close() error {
	<-s.ctx.Done()
	return s.sock.Close()
}
