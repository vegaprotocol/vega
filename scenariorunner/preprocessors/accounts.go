package preprocessors

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/proto"
)

type Accounts struct {
	ctx           context.Context
	acccountStore *storage.Account
}

func NewAccounts(ctx context.Context, accountStore *storage.Account) *Accounts {
	return &Accounts{ctx, accountStore}
}

func (a *Accounts) PreProcessors() map[core.RequestType]*core.PreProcessor {
	return map[core.RequestType]*core.PreProcessor{
		core.RequestType_ACCOUNTS_BY_PARTY:            a.accountsByParty(),
		core.RequestType_ACCOUNTS_BY_PARTY_AND_ASSET:  a.accountsByPartyAndAsset(),
		core.RequestType_ACCOUNTS_BY_PARTY_AND_MARKET: a.accountsByPartyAndMarket(),
		core.RequestType_ACCOUNTS_BY_PARTY_AND_TYPE:   a.accountsByPartyAndType(),
	}
}

func (a *Accounts) accountsByParty() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.AccountsByPartyRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				resp, err := a.acccountStore.GetByParty(req.PartyID)
				if err != nil {
					return nil, err
				}
				return &protoapi.AccountsByPartyResponse{Accounts: resp}, nil
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.AccountsByPartyRequest{},
		PreProcess:   preProcessor,
	}
}

func (a *Accounts) accountsByPartyAndAsset() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.AccountsByPartyAndAssetRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				resp, err := a.acccountStore.GetByPartyAndAsset(req.PartyID, req.Asset)
				if err != nil {
					return nil, err
				}
				return &protoapi.AccountsByPartyAndAssetResponse{Accounts: resp}, nil
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.AccountsByPartyAndAssetRequest{},
		PreProcess:   preProcessor,
	}
}

func (a *Accounts) accountsByPartyAndMarket() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.AccountsByPartyAndMarketRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				resp, err := a.acccountStore.GetByPartyAndMarket(req.PartyID, req.MarketID)
				if err != nil {
					return nil, err
				}
				return &protoapi.AccountsByPartyAndMarketResponse{Accounts: resp}, nil
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.AccountsByPartyAndMarketRequest{},
		PreProcess:   preProcessor,
	}
}

func (a *Accounts) accountsByPartyAndType() *core.PreProcessor {
	preProcessor := func(instr *core.Instruction) (*core.PreProcessedInstruction, error) {
		req := &protoapi.AccountsByPartyAndTypeRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, core.ErrInstructionInvalid
		}
		return instr.PreProcess(
			func() (proto.Message, error) {
				resp, err := a.acccountStore.GetByPartyAndType(req.PartyID, req.Type)
				if err != nil {
					return nil, err
				}
				return &protoapi.AccountsByPartyAndTypeResponse{Accounts: resp}, nil
			})
	}
	return &core.PreProcessor{
		MessageShape: &protoapi.AccountsByPartyAndTypeRequest{},
		PreProcess:   preProcessor,
	}
}
