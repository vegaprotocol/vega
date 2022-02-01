package oracles

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

const (
	BuiltinOraclePrefix    = "vegaprotocol.builtin"
	BuiltinOracleTimestamp = BuiltinOraclePrefix + ".timestamp"
)

type Builtin struct {
	log    *logging.Logger
	engine *Engine
}

func NewBuiltinOracle(engine *Engine, ts TimeService) *Builtin {
	builtinOracle := &Builtin{
		log:    logging.NewProdLogger(),
		engine: engine,
	}

	ts.NotifyOnTick(builtinOracle.BroadcastInternalTimestamp)

	return builtinOracle
}

func (b *Builtin) BroadcastInternalTimestamp(ctx context.Context, ts time.Time) {
	data := OracleData{
		PubKeys: nil,
		Data: map[string]string{
			BuiltinOracleTimestamp: fmt.Sprintf("%d", ts.UnixNano()),
		},
	}

	if err := b.engine.BroadcastData(ctx, data); err != nil {
		b.log.Error("Could not broadcast timestamp from built-in oracle", logging.Error(err))
	}
}
