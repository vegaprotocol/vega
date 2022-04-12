package entities

import (
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/types"
)

type AccountFilter struct {
	Asset        Asset
	Parties      []Party
	AccountTypes []types.AccountType
	Markets      []Market
}

func AccountFilterFromProto(pbFilter *v2.AccountFilter) (AccountFilter, error) {
	filter := AccountFilter{}
	if pbFilter != nil {
		if pbFilter.AssetId != "" {
			filter.Asset.ID = NewAssetID(pbFilter.AssetId)
		}
		for _, partyID := range pbFilter.PartyIds {
			filter.Parties = append(filter.Parties, Party{ID: NewPartyID(partyID)})
		}

		filter.AccountTypes = append(filter.AccountTypes, pbFilter.AccountTypes...)

		for _, marketID := range pbFilter.MarketIds {
			filter.Markets = append(filter.Markets, Market{ID: NewMarketID(marketID)})
		}
	}
	return filter, nil
}
