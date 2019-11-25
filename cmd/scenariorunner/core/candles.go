package core

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/proto"

	"code.vegaprotocol.io/vega/vegatime"
)

type Candles struct {
	ctx         context.Context
	candleStore *storage.Candle
}

func NewCandles(ctx context.Context, candleStore *storage.Candle) *Candles {
	return &Candles{ctx, candleStore}
}

func (c *Candles) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_CANDLES: c.candles(),
	}
}

func (c *Candles) candles() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.CandlesRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
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
	return &PreProcessor{
		MessageShape: &protoapi.CandlesRequest{},
		PreProcess:   preProcessor,
	}
}
