package genesis

import (
	"context"
	"time"

	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
)

type Handler struct {
	log *logging.Logger
	cfg Config

	onGenesisTimeLoadedCB     []func(context.Context, time.Time)
	onGenesisAppStateLoadedCB []func(context.Context, []byte) error
	onGenesisChainIDLoadedCB  []func(context.Context, string)
}

func New(log *logging.Logger, cfg Config) *Handler {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Level)
	return &Handler{
		log:                       log,
		cfg:                       cfg,
		onGenesisTimeLoadedCB:     []func(context.Context, time.Time){},
		onGenesisAppStateLoadedCB: []func(context.Context, []byte) error{},
		onGenesisChainIDLoadedCB:  []func(context.Context, string){},
	}
}

// ReloadConf update the internal configuration of the positions engine.
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

func (h *Handler) OnGenesis(ctx context.Context, t time.Time, state []byte) error {
	h.log.Debug("vega time at genesis",
		logging.String("time", t.String()))
	for _, f := range h.onGenesisTimeLoadedCB {
		f(ctx, t)
	}

	h.log.Debug("vega initial state at genesis",
		logging.String("state", string(state)))

	chainID, _ := vgcontext.ChainIDFromContext(ctx)
	h.log.Debug("chain id",
		logging.String("chainID", chainID))
	for _, f := range h.onGenesisChainIDLoadedCB {
		f(ctx, chainID)
	}
	for _, f := range h.onGenesisAppStateLoadedCB {
		if err := f(ctx, state); err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) OnGenesisTimeLoaded(f ...func(context.Context, time.Time)) {
	h.onGenesisTimeLoadedCB = append(h.onGenesisTimeLoadedCB, f...)
}

func (h *Handler) OnGenesisAppStateLoaded(f ...func(context.Context, []byte) error) {
	h.onGenesisAppStateLoadedCB = append(h.onGenesisAppStateLoadedCB, f...)
}

func (h *Handler) OnGenesisChainIDLoaded(f ...func(context.Context, string)) {
	h.onGenesisChainIDLoadedCB = append(h.onGenesisChainIDLoadedCB, f...)
}
