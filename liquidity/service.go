package liquidity

import (
	"context"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type Svc struct {
	config Config
	log    *logging.Logger
}

func NewService(log *logging.Logger, config Config) *Svc {
	log = log.Named(namedLogger)
	return &Svc{
		log:    log,
		config: config,
	}
}

// ReloadConf update the internal configuration of the order service
func (s *Svc) ReloadConf(config Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != config.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", config.Level.String()),
		)
		s.log.SetLevel(config.Level.Get())
	}

	s.config = config
}

func (s *Svc) PrepareLiquidityProvisionSubmission(_ context.Context, _ *types.LiquidityProvisionSubmission) error {
	return nil
}
