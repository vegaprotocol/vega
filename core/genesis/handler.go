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
