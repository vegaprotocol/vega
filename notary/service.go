package notary

import (
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

// Plugin ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/plugin_mock.go -package mocks code.vegaprotocol.io/vega/notary Plugin
type Plugin interface {
	GetByID(string) ([]commandspb.NodeSignature, error)
}

// Svc is governance service, responsible for managing proposals and votes.
type Svc struct {
	cfg Config
	log *logging.Logger
	p   Plugin
}

// NewService creates new governance service instance
func NewService(log *logging.Logger, cfg Config, plugin Plugin) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Svc{
		cfg: cfg,
		log: log,
		p:   plugin,
	}
}

// ReloadConf updates the internal configuration of the collateral engine
func (s *Svc) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.cfg = cfg
}

func (s *Svc) GetByID(id string) ([]commandspb.NodeSignature, error) {
	return s.p.GetByID(id)
}
