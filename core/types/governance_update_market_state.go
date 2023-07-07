package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsUpdateMarketState struct {
	UpdateMarketState *UpdateMarketState
}

type UpdateMarketState struct {
	Changes *MarketStateUpdateConfiguration
}

func (a ProposalTermsUpdateMarketState) String() string {
	return fmt.Sprintf(
		"updateMarketState(%s)",
		reflectPointerToString(a.UpdateMarketState),
	)
}

func (a ProposalTermsUpdateMarketState) IntoProto() *vegapb.ProposalTerms_UpdateMarketState {
	return &vegapb.ProposalTerms_UpdateMarketState{
		UpdateMarketState: a.UpdateMarketState.IntoProto(),
	}
}

func (a ProposalTermsUpdateMarketState) isPTerm() {}

func (a ProposalTermsUpdateMarketState) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsUpdateMarketState) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateMarketState
}

func (a ProposalTermsUpdateMarketState) DeepClone() proposalTerm {
	if a.UpdateMarketState == nil {
		return &ProposalTermsUpdateMarketState{}
	}
	return &ProposalTermsUpdateMarketState{
		UpdateMarketState: a.UpdateMarketState.DeepClone(),
	}
}

func NewTerminateMarketFromProto(p *vegapb.ProposalTerms_UpdateMarketState) (*ProposalTermsUpdateMarketState, error) {
	var terminateMarket *UpdateMarketState
	if p.UpdateMarketState != nil {
		terminateMarket = &UpdateMarketState{}

		var price *num.Uint
		if p.UpdateMarketState.Changes.Price != nil {
			price, _ = num.UintFromString(*p.UpdateMarketState.Changes.Price, 10)
		}

		if p.UpdateMarketState.Changes != nil {
			terminateMarket.Changes = &MarketStateUpdateConfiguration{
				MarketID:        p.UpdateMarketState.Changes.MarketId,
				UpdateType:      p.UpdateMarketState.Changes.UpdateType,
				SettlementPrice: price,
			}
		}
	}

	return &ProposalTermsUpdateMarketState{
		UpdateMarketState: terminateMarket,
	}, nil
}

func (c UpdateMarketState) IntoProto() *vegapb.UpdateMarketState {
	var price *string
	if c.Changes.SettlementPrice != nil {
		pp := c.Changes.SettlementPrice.String()
		price = &pp
	}
	return &vegapb.UpdateMarketState{
		Changes: &vegapb.UpdateMarketStateConfiguration{
			MarketId:   c.Changes.MarketID,
			UpdateType: c.Changes.UpdateType,
			Price:      price,
		},
	}
}

func (c UpdateMarketState) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		reflectPointerToString(c.Changes),
	)
}

func (c UpdateMarketState) DeepClone() *UpdateMarketState {
	price := c.Changes.SettlementPrice
	if price != nil {
		price = price.Clone()
	}
	return &UpdateMarketState{
		Changes: &MarketStateUpdateConfiguration{
			MarketID:        c.Changes.MarketID,
			UpdateType:      c.Changes.UpdateType,
			SettlementPrice: price,
		},
	}
}

type MarketStateUpdateType = vegapb.MarketStateUpdateType

const (
	MarketStateUpdateTypeUnspecified MarketStateUpdateType = vegapb.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_UNSPECIFIED
	MarketStateUpdateTypeTerminate   MarketStateUpdateType = vegapb.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_TERMINATE
	MarketStateUpdateTypeSuspend     MarketStateUpdateType = vegapb.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_SUSPEND
	MarketStateUpdateTypeResume      MarketStateUpdateType = vegapb.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_RESUME
)

type MarketStateUpdateConfiguration struct {
	MarketID        string
	SettlementPrice *num.Uint
	UpdateType      MarketStateUpdateType
}

func (c MarketStateUpdateConfiguration) String() string {
	return fmt.Sprintf("marketID(%s), updateType(%d), settlementPrice(%s)", c.MarketID, c.UpdateType, reflectPointerToString(c.SettlementPrice))
}
