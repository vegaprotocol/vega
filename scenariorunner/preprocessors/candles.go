package preprocessors

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/storage"

	"code.vegaprotocol.io/vega/vegatime"
	"github.com/golang/protobuf/proto"
)

type Candles struct {
	ctx         context.Context
	candleStore *storage.Candle
}

func NewCandles(ctx context.Context, candleStore *storage.Candle) *Candles {
	return &Candles{ctx, candleStore}
}

func (c *Candles) PreProcessors() map[core.RequestType]*core.PreProcessor {
	return map[core.RequestType]*core.PreProcessor{
		core.RequestType_CANDLES: c.candles(),
	}
}

func (c *Candles) candles() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.CandlesRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				resp, err := c.candleStore.GetCandles(c.ctx, req.MarketID, vegatime.UnixNano(req.SinceTimestamp), req.Interval)
				if err != nil {
					return nil, err
				}
				return &protoapi.CandlesResponse{Candles: resp}, nil
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.CandlesRequest{},
		PreProcess:   preProcessor,
	}
}
