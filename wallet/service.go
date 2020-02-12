package wallet

import (
	"context"
	"fmt"
	"net/http"

	"code.vegaprotocol.io/vega/logging"
	"github.com/rs/cors"
)

type Service struct {
	cfg *Config
	log *logging.Logger
	s   *http.Server
}

func NewService(log *logging.Logger, cfg *Config) *Service {
	return &Service{
		log: log,
		cfg: cfg,
	}
}

func (s *Service) Start() error {
	handler := newHandler(s.log)

	s.s = &http.Server{
		Addr:    fmt.Sprintf("%s:%v", s.cfg.IP, s.cfg.Port),
		Handler: cors.AllowAll().Handler(handler), // middlewar with cors
	}

	s.log.Info("starting wallet http server", logging.String("address", s.s.Addr))
	return s.s.ListenAndServe()
}

func (s *Service) Stop() error {
	return s.s.Shutdown(context.Background())
}
