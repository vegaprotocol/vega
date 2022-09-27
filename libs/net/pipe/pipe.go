// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package pipe

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
)

type Pipe struct {
	ch   chan net.Conn
	done chan any
	addr Addr
}

func NewPipe(name string) *Pipe {
	return &Pipe{
		ch:   make(chan net.Conn),
		addr: Addr{name: name},
	}
}

func (p *Pipe) Accept() (net.Conn, error) {
	conn, ok := <-p.ch
	if !ok {
		return nil, fmt.Errorf("connection channel unexpectedly closed")
	}
	return conn, nil
}

func (p *Pipe) Close() error {
	close(p.ch)
	<-p.done
	return nil
}

func (p *Pipe) Addr() net.Addr {
	return &p.addr
}

func (p *Pipe) Dial(ctx context.Context, something string) (net.Conn, error) {
	conn1, conn2 := net.Pipe()
	select {
	case p.ch <- conn1:
		return conn2, nil
	case <-ctx.Done():
		conn1.Close()
		conn2.Close()
		return nil, ctx.Err()
	}
}

func (p *Pipe) DialGRPC(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	newOpts := make([]grpc.DialOption, 0, len(opts)+1)
	newOpts = append(newOpts, grpc.WithContextDialer(p.Dial))
	newOpts = append(newOpts, opts...)

	return grpc.DialContext(
		context.Background(),
		p.Addr().String(),
		newOpts...,
	)
}
