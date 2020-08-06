package genesis

import (
	"time"

	"code.vegaprotocol.io/vega/logging"
)

type Handler struct {
	log *logging.Logger
	cfg Config

	onGenesisTimeLoadedCB     []func(time.Time)
	onGenesisAppStateLoadedCB []func([]byte) error
}

func New(log *logging.Logger, cfg Config) *Handler {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Level)
	return &Handler{
		log:                       log,
		cfg:                       cfg,
		onGenesisTimeLoadedCB:     []func(time.Time){},
		onGenesisAppStateLoadedCB: [](func([]byte) error){},
	}
}

// ReloadConf update the internal configuration of the positions engine
func (h *Handler) ReloadConf(cfg Config) {
	h.log.Info("reloading configuration")
	if h.log.GetLevel() != cfg.Level.Get() {
		h.log.Info("updating log level",
			logging.String("old", h.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		h.log.SetLevel(cfg.Level.Get())
	}
	h.cfg = cfg
}

func (h *Handler) OnGenesis(t time.Time, state []byte, validatorsPubkey [][]byte) error {
	h.log.Debug("vega time at genesis",
		logging.String("time", t.String()))
	for _, f := range h.onGenesisTimeLoadedCB {
		f(t)
	}

	h.log.Debug("vega initial state at genesis",
		logging.String("state", string(state)))
	for _, f := range h.onGenesisAppStateLoadedCB {
		if err := f(state); err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) OnGenesisTimeLoaded(f func(time.Time)) {
	h.onGenesisTimeLoadedCB = append(h.onGenesisTimeLoadedCB, f)
}

func (h *Handler) OnGenesisAppStateLoaded(f func([]byte) error) {
	h.onGenesisAppStateLoadedCB = append(h.onGenesisAppStateLoadedCB, f)
}
