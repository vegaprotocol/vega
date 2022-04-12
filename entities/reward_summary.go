package entities

import (
	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type RewardSummary struct {
	PartyID PartyID
	AssetID AssetID
	Amount  decimal.Decimal
}

func (r *RewardSummary) ToProto() *vega.RewardSummary {
	protoRewardSummary := vega.RewardSummary{
		PartyId: r.PartyID.String(),
		AssetId: r.AssetID.String(),
		Amount:  r.Amount.String(),
	}
	return &protoRewardSummary
}
