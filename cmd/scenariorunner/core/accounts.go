package core

import (
	"context"

	protoapi "code.vegaprotocol.io/vega/proto/api"
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

func (a *Accounts) PreProcessors() map[RequestType]*PreProcessor {
	return map[RequestType]*PreProcessor{
		RequestType_ACCOUNTS_BY_PARTY_AND_ASSET:  a.accountsByPartyAndAsset(),
		RequestType_ACCOUNTS_BY_PARTY_AND_MARKET: a.accountsByPartyAndMarket(),
		RequestType_ACCOUNTS_BY_PARTY_AND_TYPE:   a.accountsByPartyAndType(),
	}
}

func (a *Accounts) accountsByPartyAndAsset() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.AccountsByPartyAndAssetRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
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
	return &PreProcessor{
		MessageShape: &protoapi.AccountsByPartyAndAssetRequest{},
		PreProcess:   preProcessor,
	}
}

func (a *Accounts) accountsByPartyAndMarket() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.AccountsByPartyAndMarketRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
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
	return &PreProcessor{
		MessageShape: &protoapi.AccountsByPartyAndMarketRequest{},
		PreProcess:   preProcessor,
	}
}

func (a *Accounts) accountsByPartyAndType() *PreProcessor {
	preProcessor := func(instr *Instruction) (*PreProcessedInstruction, error) {
		req := &protoapi.AccountsByPartyAndTypeRequest{}
		if err := proto.Unmarshal(instr.Message.Value, req); err != nil {
			return nil, ErrInstructionInvalid
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
	return &PreProcessor{
		MessageShape: &protoapi.AccountsByPartyAndTypeRequest{},
		PreProcess:   preProcessor,
	}
}
