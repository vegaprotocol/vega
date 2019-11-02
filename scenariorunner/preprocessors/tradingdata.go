package preprocessors

import (
	"context"

	"code.vegaprotocol.io/vega/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
)

type TradingData struct {
	mappings map[string]*core.PreProcessor
}

func NewTradingData(ctx context.Context, t *api.TradingDataService) (*TradingData, error) {

	m := map[string]*core.PreProcessor{
		// orders
		"ordersbymarket":     ordersByMarket(ctx, t),
		"ordersbyparty":      ordersByParty(ctx, t),
		"orderbymarketandid": orderByMarketAndID(ctx, t),
		"orderbyreference":   orderByReference(ctx, t),
		// markets
		"marketbyid":  marketByID(ctx, t),
		"markets":     markets(ctx, t),
		"marketdepth": marketDepth(ctx, t),
		"lasttrade":   lastTrade(ctx, t),
		// parties
		"partybyid": partyByID(ctx, t),
		"parties":   parties(ctx, t),

		// trades
		"tradesbymarket": tradesByMarket(ctx, t),
		"tradesbyparty":  tradesByParty(ctx, t),
		"tradesbyorder":  tradesByOrder(ctx, t),

		// positions
		"positionsbyparty": positionsByParty(ctx, t),

		// candles
		"candles": candles(ctx, t),

		// metrics
		"getvegatime": getVegaTime(ctx, t),

		// accounts
		"accountsbyparty":          accountsByParty(ctx, t),
		"accountsbypartyandmarket": accountsByPartyAndMarket(ctx, t),
		"accountsbypartyandtype":   accountsByPartyAndType(ctx, t),
		"accountsbypartyandasset":  accountsByPartyAndAsset(ctx, t),
	}

	return &TradingData{m}, nil
}

func (e *Execution) TradingData() map[string]*core.PreProcessor {
	return e.mappings
}

// orders
func ordersByMarket(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.OrdersByMarketRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.OrdersByMarket(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func ordersByParty(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.OrdersByPartyRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.OrdersByParty(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func orderByMarketAndID(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.OrderByMarketAndIdRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.OrderByMarketAndId(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}
func orderByReference(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.OrderByReferenceRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.OrderByReference(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

// markets
func marketByID(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.MarketByIDRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.MarketByID(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func markets(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &empty.Empty{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.Markets(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func marketDepth(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.MarketDepthRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.MarketDepth(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func lastTrade(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.LastTradeRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.LastTrade(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

// parties
func partyByID(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.PartyByIDRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.PartyByID(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func parties(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &empty.Empty{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.Parties(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

// trades
func tradesByMarket(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.TradesByMarketRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.TradesByMarket(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func tradesByParty(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.TradesByPartyRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.TradesByParty(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func tradesByOrder(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.TradesByOrderRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.TradesByOrder(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

// positions
func positionsByParty(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.PositionsByPartyRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.PositionsByParty(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

// candles
func candles(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.CandlesRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.Candles(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

// metrics

/*rpc Statistics(google.protobuf.Empty) returns (vega.Statistics);
  rpc GetVegaTime(google.protobuf.Empty) returns (VegaTimeResponse);*/
func getVegaTime(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &empty.Empty{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.GetVegaTime(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

// accounts
func accountsByParty(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.AccountsByPartyRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.AccountsByParty(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func accountsByPartyAndMarket(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.AccountsByPartyAndMarketRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.AccountsByPartyAndMarket(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func accountsByPartyAndType(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.AccountsByPartyAndTypeRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.AccountsByPartyAndType(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}

func accountsByPartyAndAsset(ctx context.Context, t *api.TradingDataService) *core.PreProcessor {
	req := &protoapi.AccountsByPartyAndAssetRequest{}
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) { return t.AccountsByPartyAndAsset(ctx, req) })
	}
	return &core.PreProcessor{
		MessageShape: req,
		PreProcess:   preProcessor,
	}
}
