// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package pipe

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
)

type Pipe struct {
	ch   chan net.Conn
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
	return nil
}

func (p *Pipe) Addr() net.Addr {
	return &p.addr
}

func (p *Pipe) Dial(ctx context.Context, _ string) (net.Conn, error) {
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

func (p *Pipe) DialGRPC(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	newOpts := make([]grpc.DialOption, 0, len(opts)+1)
	newOpts = append(newOpts, grpc.WithContextDialer(p.Dial))
	newOpts = append(newOpts, opts...)

	return grpc.DialContext(ctx, p.Addr().String(), newOpts...)
}
