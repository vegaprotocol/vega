package entities

import (
	"encoding/json"
	"fmt"

	"code.vegaprotocol.io/vega/protos/vega"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type LiquidityProvider struct {
	PartyID    PartyID
	MarketID   MarketID
	Ordinality int64
	FeeShare   *vega.LiquidityProviderFeeShare
}

type LiquidityProviderCursor struct {
	MarketID   MarketID `json:"marketId"`
	PartyID    PartyID  `json:"partyId"`
	Ordinality int64    `json:"ordinality"`
}

func (c LiquidityProviderCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal liquidity provision cursor: %w", err))
	}
	return string(bs)
}

func (c *LiquidityProviderCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), c)
}

func (lp LiquidityProvider) ToProto() *v2.LiquidityProvider {
	return &v2.LiquidityProvider{
		PartyId:  lp.PartyID.String(),
		MarketId: lp.MarketID.String(),
		FeeShare: lp.FeeShare,
	}
}

func (lp LiquidityProvider) Cursor() *Cursor {
	c := LiquidityProviderCursor{
		PartyID: lp.PartyID,
	}

	return NewCursor(c.String())
}

func (lp LiquidityProvider) ToProtoEdge(...any) (*v2.LiquidityProviderEdge, error) {
	return &v2.LiquidityProviderEdge{
		Node:   lp.ToProto(),
		Cursor: lp.Cursor().Encode(),
	}, nil
}
