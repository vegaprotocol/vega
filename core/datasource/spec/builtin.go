// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
