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

package api

import (
	"fmt"
	"net"

	"github.com/soheilhy/cmux"

	"code.vegaprotocol.io/vega/logging"
)

type Portal struct {
	Config
	log             *logging.Logger
	mux             cmux.CMux
	portalListener  net.Listener
	grpcListener    net.Listener
	gatewayListener net.Listener
}

func NewPortal(config Config, log *logging.Logger) *Portal {
	log = log.Named(portalNamedLogger)

	address := net.JoinHostPort(config.ListenAddress, fmt.Sprintf("%v", config.ListenPort))
	portalListener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("unable to start portal listener", logging.String("address", address), logging.Error(err))
	}

	mux := cmux.New(portalListener)
	grpcListener := mux.Match(cmux.HTTP2())
	gatewayListener := mux.Match(cmux.HTTP1Fast())

	portal := Portal{
		Config:          config,
		log:             log,
		mux:             mux,
		portalListener:  portalListener,
		grpcListener:    grpcListener,
		gatewayListener: gatewayListener,
	}
	return &portal
}

func (p *Portal) Serve() error {
	p.log.Info("Starting portal")
	return p.mux.Serve()
}

func (p *Portal) GatewayListener() net.Listener {
	return p.gatewayListener
}

func (p *Portal) GRPCListener() net.Listener {
	return p.grpcListener
}

func (p *Portal) Stop() {
	p.log.Info("Stopping portal")
	_ = p.gatewayListener.Close()
	_ = p.grpcListener.Close()
	_ = p.portalListener.Close()
}
