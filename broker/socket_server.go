package broker

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/data-node/proto/events/v1"

	"github.com/golang/protobuf/proto"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
)

// SocketServer receives events from a remote broker.
// This is used by the data node to receive events from a non-validating core node.
type SocketServer struct {
	log  *logging.Logger
	sock protocol.Socket
}

func NewSocketReceiver(log *logging.Logger, config *SocketConfig) (*SocketServer, error) {
	sock, err := pull.NewSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to create new socket: %w", err)
	}

	addr := fmt.Sprintf("tcp://%s:%d", config.IP, config.Port)

	if err = sock.Listen(addr); err != nil {
		return nil, fmt.Errorf("failed to listen on %v: %w", addr, err)
	}

	return &SocketServer{
		log:  log,
		sock: sock,
	}, nil

}

func (s SocketServer) Receive(ctx context.Context, ch chan events.Event) {
	var err error
	var msg []byte

	for {
		msg, err = s.sock.Recv()
		if err != nil {
			s.log.Error("failed to receive message", logging.Error(err))
		}

		var be eventspb.BusEvent
		err = proto.Unmarshal(msg, &be)
		if err != nil {
			s.log.Error("failed to receive message", logging.Error(err))
		}

		evt, _ := be.GetEvent().(events.Event)
		ch <- evt
	}
}

func (s SocketServer) Close() error {
	return s.sock.Close()
}
