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

package genesis

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

type Handler struct {
	log *logging.Logger
	cfg Config

	onGenesisTimeLoadedCB     []func(context.Context, time.Time)
	onGenesisAppStateLoadedCB []func(context.Context, []byte) error
}

func New(log *logging.Logger, cfg Config) *Handler {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Level)
	return &Handler{
		log:                       log,
		cfg:                       cfg,
		onGenesisTimeLoadedCB:     []func(context.Context, time.Time){},
		onGenesisAppStateLoadedCB: []func(context.Context, []byte) error{},
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
