package entities

import (
	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type RewardSummary struct {
	PartyID []byte
	AssetID []byte
	Amount  decimal.Decimal
}

func (r *RewardSummary) PartyHexID() string {
	return Party{ID: r.PartyID}.HexID()
}

func (r *RewardSummary) AssetHexID() string {
	return Asset{ID: r.AssetID}.HexID()
}

func (r *RewardSummary) ToProto() *vega.RewardSummary {
	protoRewardSummary := vega.RewardSummary{
		PartyId: r.PartyHexID(),
		AssetId: r.AssetHexID(),
		Amount:  r.Amount.String(),
	}
	return &protoRewardSummary
}
