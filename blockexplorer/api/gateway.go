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
	_ "embed"
	"fmt"
	"net"
	"net/http"

	"code.vegaprotocol.io/vega/logging"
)

//go:embed gateway.css
var indexCSS string

type Gateway struct {
	GatewayConfig
	httpServerMux *http.ServeMux
	log           *logging.Logger
	handlers      map[string]GatewayHandler
}

type GatewayHandler interface {
	http.Handler
	Name() string
	Description() string
	Start() error
}

func NewGateway(log *logging.Logger, config GatewayConfig) *Gateway {
	log = log.Named(gatewayNamedLogger)
	return &Gateway{
		GatewayConfig: config,
		httpServerMux: http.NewServeMux(),
		log:           log,
		handlers:      make(map[string]GatewayHandler),
	}
}

func (s *Gateway) Register(handler GatewayHandler, endpoint string) {
	s.log.Info("registered with api gateway", logging.String("endpoint", endpoint), logging.String("handler", handler.Name()))
	s.handlers[endpoint] = handler
	s.httpServerMux.Handle(endpoint+"/", http.StripPrefix(endpoint, handler))
}

func (s *Gateway) Serve(lis net.Listener) error {
	logAddr := logging.String("address", lis.Addr().String())
	s.log.Info("gateway starting", logAddr)
	defer s.log.Info("gateway stopping", logAddr)
	return http.Serve(lis, s.httpServerMux)
}

func (s *Gateway) StartHandlers() error {
	s.httpServerMux.HandleFunc("/", s.indexHandler)
	for _, handler := range s.handlers {
		if err := handler.Start(); err != nil {
			s.log.Error("error starting",
				logging.String("handler", handler.Name()),
				logging.Error(err))
		}
	}
	return nil
}

func (s *Gateway) indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<html><body>`)
	fmt.Fprintf(w, `<style>%s</style>`, indexCSS)
	fmt.Fprintf(w, `<header><h1>Block Explorer API Gateway</h1></header>`)
	fmt.Fprintf(w, "<main>")
	for endpoint, handler := range s.handlers {
		fmt.Fprintf(w, "<p><a href='%s'>%s</a> - %s</p>", endpoint, endpoint, handler.Description())
	}
	fmt.Fprintf(w, "</main>")
	fmt.Fprintf(w, "</html></body>")
}
