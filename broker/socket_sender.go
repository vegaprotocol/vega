package broker

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"code.vegaprotocol.io/vega/logging"

	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
)

const (
	dialRetryInterval = 5 * time.Second
)

// SocketSender sends events sent to this broker over a socket to a remote broker.
// This is used to stream events from a non-validating core node to a data node.
// The sender will try to connect to the receiver address indefinitely.
type SocketSender struct {
	log  *logging.Logger
	sock protocol.Socket
}

func NewSocketSender(log *logging.Logger, config *SocketConfig) (*SocketSender, error) {
	var sock mangos.Socket
	var err error

	if !config.Enabled {
		return &SocketSender{}, nil
	}

	if sock, err = push.NewSocket(); err != nil {
		return nil, fmt.Errorf("failed to create new socket: %w", err)
	}

	addr := fmt.Sprintf("tcp://127.0.0.1:%d", config.Port)
RETRY_LOOP:
	for {
		if err = sock.Dial(addr); err != nil {
			log.Error(fmt.Sprintf("failed to connect to %v, retrying", addr), logging.Error(err))
			time.Sleep(dialRetryInterval)
			continue RETRY_LOOP
		}

		return &SocketSender{
			log:  log,
			sock: sock,
		}, nil
	}
}

func (s SocketSender) Send(r io.Reader) error {
	buf, _ := ioutil.ReadAll(r)
	if err := s.sock.Send(buf); err != nil {
		return fmt.Errorf("failed to send on socket: %w", err)
	}
	return nil
}

func (s SocketSender) Close() error {
	return s.sock.Close()
}
