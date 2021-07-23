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

// socketClient sends events sent to this broker over a socket to a remote broker.
// This is used to stream events from a non-validating core node to a data node.
// The sender will try to connect to the receiver address indefinitely.
type socketClient struct {
	log  *logging.Logger
	sock protocol.Socket
}

func NewSocketClient(log *logging.Logger, config *SocketConfig) (SocketClient, error) {
	var sock mangos.Socket
	var err error

	if !config.Enabled {
		return &socketClient{}, nil
	}

	if sock, err = push.NewSocket(); err != nil {
		return nil, fmt.Errorf("failed to create new socket: %w", err)
	}

	addr := fmt.Sprintf("tcp://%s:%d", config.IP, config.Port)
	ticker := time.NewTicker(dialRetryInterval)
	for range ticker.C {
		if err = sock.Dial(addr); err != nil {
			log.Error(fmt.Sprintf("failed to connect to %v, retrying", addr), logging.Error(err))
			continue
		}
	}

	return &socketClient{
		log:  log,
		sock: sock,
	}, nil
}

func (s socketClient) Send(r io.Reader) error {
	buf, _ := ioutil.ReadAll(r)
	if err := s.sock.Send(buf); err != nil {
		return fmt.Errorf("failed to send on socket: %w", err)
	}
	return nil
}

func (s socketClient) Close() error {
	return s.sock.Close()
}
