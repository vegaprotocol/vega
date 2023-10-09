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

package spec

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/logging"
)

const (
	BuiltinPrefix      = "vegaprotocol.builtin"
	BuiltinTimestamp   = BuiltinPrefix + ".timestamp"
	BuiltinTimeTrigger = BuiltinPrefix + ".timetrigger"
)

type Builtin struct {
	log    *logging.Logger
	engine *Engine
}

func NewBuiltin(engine *Engine, ts TimeService) *Builtin {
	builtinOracle := &Builtin{
		log:    logging.NewProdLogger(),
		engine: engine,
	}

	return builtinOracle
}

func (b *Builtin) OnTick(ctx context.Context, _ time.Time) {
	data := common.Data{
		Signers: nil,
		Data: map[string]string{
			BuiltinTimestamp:   fmt.Sprintf("%d", b.engine.timeService.GetTimeNow().Unix()),
			BuiltinTimeTrigger: fmt.Sprintf("%d", b.engine.timeService.GetTimeNow().Unix()),
		},
	}

	if err := b.engine.BroadcastData(ctx, data); err != nil {
		b.log.Error("Could not broadcast timestamp from built-in oracle", logging.Error(err))
	}
}
