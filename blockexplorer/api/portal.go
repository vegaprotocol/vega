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

package api

import (
	"fmt"
	"net"

	"code.vegaprotocol.io/vega/logging"
	"github.com/soheilhy/cmux"
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
	p.log.Info("portal started")
	defer p.log.Info("portal finished")
	return p.mux.Serve()
}

func (p *Portal) GatewayListener() net.Listener {
	return p.gatewayListener
}

func (p *Portal) GRPCListener() net.Listener {
	return p.grpcListener
}
