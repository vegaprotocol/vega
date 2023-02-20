// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"golang.org/x/sync/errgroup"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/protobuf/proto"
	"go.nanomsg.org/mangos/v3"
	mangosErr "go.nanomsg.org/mangos/v3/errors"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pair"
	_ "go.nanomsg.org/mangos/v3/transport/inproc" // changes behavior of nanomsg
	_ "go.nanomsg.org/mangos/v3/transport/tcp"    // changes behavior of nanomsg
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
	sock, err := pair.NewSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to create new socket: %w", err)
	}

	return &socketServer{
		log:    log.Named("socket-server"),
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

	// listenOptions := map[string]interface{}{mangos.OptionMaxRecvSize: 0}
	listenOptions := map[string]interface{}{}

	listener, err := s.sock.NewListener(addr, listenOptions)
	if err != nil {
		return fmt.Errorf("failed to make listener %w", err)
	}

	if err := listener.Listen(); err != nil {
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

func (s socketServer) Receive(ctx context.Context) (<-chan []byte, <-chan error) {
	outboundCh := make(chan []byte, s.config.SocketServerOutboundBufferSize)
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
			// Listen for context cancels, even if we're blocked sending events
			select {
			case outboundCh <- msg:
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

func (s socketServer) Send(evt events.Event) error {
	msg, err := proto.Marshal(evt.StreamMessage())
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = s.sock.Send(msg)
	if err != nil {
		switch err {
		case protocol.ErrClosed:
			return fmt.Errorf("socket is closed: %w", err)
		case protocol.ErrSendTimeout:
			return fmt.Errorf("failed to queue message on socket: %w", err)
		default:
			return fmt.Errorf("failed to send to socket: %w", err)
		}
	}

	return nil
}

func (s socketServer) close() error {
	s.log.Info("Closing socket server")
	return s.sock.Close()
}
